package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	maxPacket   = 64 * 1024
	retryEvery  = 80 * time.Millisecond
	idlePeerTTL = 5 * time.Minute
)

// peer talks to one UDP remote. Streams are stop-and-wait so SOCKS
// over lossy UDP still roughly works without a real reliability stack.
type peer struct {
	conn   *net.UDPConn
	remote *net.UDPAddr
	key    *[32]byte

	mu      sync.Mutex
	streams map[uint32]*stream
	nextID  uint32

	writeMu sync.Mutex
}

type stream struct {
	id     uint32
	peer   *peer
	inbox  chan []byte
	closed chan struct{}

	sendMu   sync.Mutex
	nextSeq  uint32
	pending  []byte
	pendSeq  uint32
	ackCh    chan struct{}
	waitOpen chan error

	recvMu   sync.Mutex
	recvNext uint32 // next expected data seq; duplicates only re-ACK
}

func newPeer(conn *net.UDPConn, remote *net.UDPAddr, key *[32]byte) *peer {
	return &peer{
		conn:    conn,
		remote:  remote,
		key:     key,
		streams: make(map[uint32]*stream),
		nextID:  1,
	}
}

func (p *peer) sendMsg(m message) error {
	plain := encodeMsg(m)
	pkt, err := seal(p.key, plain)
	if err != nil {
		return err
	}
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	_, err = p.conn.WriteToUDP(pkt, p.remote)
	return err
}

func (p *peer) handle(m message) {
	switch m.Type {
	case msgOpen:
		p.onOpen(m)
	case msgOpenAck, msgOpenErr:
		p.onOpenReply(m)
	case msgData:
		p.onData(m)
	case msgAck:
		p.onAck(m)
	case msgClose:
		p.onClose(m)
	}
}

func (p *peer) getStream(id uint32) *stream {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.streams[id]
}

func (p *peer) addStream(s *stream) {
	p.mu.Lock()
	p.streams[s.id] = s
	p.mu.Unlock()
}

func (p *peer) removeStream(id uint32) {
	p.mu.Lock()
	if s, ok := p.streams[id]; ok {
		delete(p.streams, id)
		select {
		case <-s.closed:
		default:
			close(s.closed)
		}
	}
	p.mu.Unlock()
}

func (p *peer) allocStream() *stream {
	p.mu.Lock()
	id := p.nextID
	p.nextID++
	s := &stream{
		id:       id,
		peer:     p,
		inbox:    make(chan []byte, 32),
		closed:   make(chan struct{}),
		ackCh:    make(chan struct{}, 1),
		waitOpen: make(chan error, 1),
	}
	p.streams[id] = s
	p.mu.Unlock()
	return s
}

func (p *peer) onOpen(m message) {
	target := string(m.Body)
	s := &stream{
		id:       m.StreamID,
		peer:     p,
		inbox:    make(chan []byte, 32),
		closed:   make(chan struct{}),
		ackCh:    make(chan struct{}, 1),
		waitOpen: make(chan error, 1),
	}
	p.addStream(s)

	go func() {
		conn, err := net.DialTimeout("tcp", target, 10*time.Second)
		if err != nil {
			_ = p.sendMsg(message{Type: msgOpenErr, StreamID: s.id, Body: []byte(err.Error())})
			p.removeStream(s.id)
			return
		}
		_ = p.sendMsg(message{Type: msgOpenAck, StreamID: s.id})
		relayStream(s, conn)
	}()
}

func (p *peer) onOpenReply(m message) {
	s := p.getStream(m.StreamID)
	if s == nil || s.waitOpen == nil {
		return
	}
	var err error
	if m.Type == msgOpenErr {
		err = fmt.Errorf("remote: %s", string(m.Body))
	}
	select {
	case s.waitOpen <- err:
	default:
	}
}

func (p *peer) onData(m message) {
	s := p.getStream(m.StreamID)
	if s == nil {
		return
	}
	_ = p.sendMsg(message{Type: msgAck, StreamID: m.StreamID, Seq: m.Seq})
	s.recvMu.Lock()
	deliver := m.Seq == s.recvNext
	if deliver {
		s.recvNext++
	}
	s.recvMu.Unlock()
	if !deliver {
		return
	}
	select {
	case s.inbox <- m.Body:
	case <-s.closed:
	}
}

func (p *peer) onAck(m message) {
	s := p.getStream(m.StreamID)
	if s == nil {
		return
	}
	s.sendMu.Lock()
	match := s.pending != nil && s.pendSeq == m.Seq
	if match {
		s.pending = nil
	}
	s.sendMu.Unlock()
	if match {
		select {
		case s.ackCh <- struct{}{}:
		default:
		}
	}
}

func (p *peer) onClose(m message) {
	p.removeStream(m.StreamID)
}

func (s *stream) writeReliable(b []byte) error {
	s.sendMu.Lock()
	seq := s.nextSeq
	s.nextSeq++
	s.pending = append([]byte(nil), b...)
	s.pendSeq = seq
	s.sendMu.Unlock()

	msg := message{Type: msgData, StreamID: s.id, Seq: seq, Body: b}
	deadline := time.After(30 * time.Second)
	for {
		if err := s.peer.sendMsg(msg); err != nil {
			return err
		}
		select {
		case <-s.ackCh:
			return nil
		case <-s.closed:
			return io.ErrClosedPipe
		case <-deadline:
			return fmt.Errorf("cloak: send timeout stream %d", s.id)
		case <-time.After(retryEvery):
		}
	}
}

func (s *stream) open(target string) error {
	if err := s.peer.sendMsg(message{Type: msgOpen, StreamID: s.id, Body: []byte(target)}); err != nil {
		return err
	}
	select {
	case err := <-s.waitOpen:
		return err
	case <-time.After(15 * time.Second):
		return fmt.Errorf("cloak: open timeout for %s", target)
	}
}

func (s *stream) closeRemote() {
	_ = s.peer.sendMsg(message{Type: msgClose, StreamID: s.id})
	s.peer.removeStream(s.id)
}

func relayStream(s *stream, conn net.Conn) {
	defer func() {
		conn.Close()
		s.closeRemote()
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		buf := make([]byte, 16*1024)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				if werr := s.writeReliable(buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		defer conn.Close()
		for {
			select {
			case <-s.closed:
				return
			case chunk, ok := <-s.inbox:
				if !ok {
					return
				}
				if _, err := conn.Write(chunk); err != nil {
					return
				}
			}
		}
	}()

	wg.Wait()
}

func readLoop(conn *net.UDPConn, key *[32]byte, resolve func(*net.UDPAddr) *peer) {
	buf := make([]byte, maxPacket)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		plain, err := open(key, buf[:n])
		if err != nil {
			continue
		}
		msg, err := decodeMsg(plain)
		if err != nil {
			continue
		}
		p := resolve(addr)
		if p == nil {
			continue
		}
		p.handle(msg)
	}
}

// server tracks clients by UDP address.
type server struct {
	conn *net.UDPConn
	key  *[32]byte

	mu    sync.Mutex
	peers map[string]*peerEntry
}

type peerEntry struct {
	peer     *peer
	lastSeen atomic.Int64
}

func runSOCKSServer(listen, keyStr string) error {
	key, err := parseKey(keyStr)
	if err != nil {
		return err
	}
	addr, err := net.ResolveUDPAddr("udp", listen)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	s := &server{
		conn:  conn,
		key:   key,
		peers: make(map[string]*peerEntry),
	}
	log.Printf("cloak socks server listening on %s", conn.LocalAddr())
	go s.reap()
	readLoop(conn, key, s.resolve)
	return nil
}

func (s *server) resolve(addr *net.UDPAddr) *peer {
	k := addr.String()
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.peers[k]
	if !ok {
		e = &peerEntry{peer: newPeer(s.conn, addr, s.key)}
		s.peers[k] = e
		log.Printf("peer %s", k)
	}
	e.lastSeen.Store(time.Now().UnixNano())
	return e.peer
}

func (s *server) reap() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for range t.C {
		cut := time.Now().Add(-idlePeerTTL).UnixNano()
		s.mu.Lock()
		for k, e := range s.peers {
			if e.lastSeen.Load() < cut {
				delete(s.peers, k)
				log.Printf("drop idle peer %s", k)
			}
		}
		s.mu.Unlock()
	}
}

func runSOCKSClient(serverAddr, keyStr, socksAddr string) error {
	key, err := parseKey(keyStr)
	if err != nil {
		return err
	}
	raddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return err
	}
	defer conn.Close()

	p := newPeer(conn, raddr, key)
	go readLoop(conn, key, func(addr *net.UDPAddr) *peer {
		if addr.String() != raddr.String() {
			return nil
		}
		return p
	})

	log.Printf("cloak socks client -> %s, socks5 on %s", serverAddr, socksAddr)
	return listenSOCKS(socksAddr, p)
}

// dialThrough opens a tunneled TCP stream to host:port via the peer.
func dialThrough(p *peer, host string, port uint16) (net.Conn, error) {
	target := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	s := p.allocStream()
	if err := s.open(target); err != nil {
		p.removeStream(s.id)
		return nil, err
	}
	return &streamConn{stream: s}, nil
}

type streamConn struct {
	stream *stream
	rbuf   []byte
}

func (c *streamConn) Read(b []byte) (int, error) {
	if len(c.rbuf) > 0 {
		n := copy(b, c.rbuf)
		c.rbuf = c.rbuf[n:]
		return n, nil
	}
	select {
	case <-c.stream.closed:
		return 0, io.EOF
	case chunk, ok := <-c.stream.inbox:
		if !ok {
			return 0, io.EOF
		}
		n := copy(b, chunk)
		if n < len(chunk) {
			c.rbuf = chunk[n:]
		}
		return n, nil
	}
}

func (c *streamConn) Write(b []byte) (int, error) {
	if err := c.stream.writeReliable(b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *streamConn) Close() error {
	c.stream.closeRemote()
	return nil
}

func (c *streamConn) LocalAddr() net.Addr                { return stubAddr("cloak-local") }
func (c *streamConn) RemoteAddr() net.Addr               { return stubAddr("cloak-remote") }
func (c *streamConn) SetDeadline(t time.Time) error      { return nil }
func (c *streamConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *streamConn) SetWriteDeadline(t time.Time) error { return nil }

type stubAddr string

func (a stubAddr) Network() string { return "cloak" }
func (a stubAddr) String() string  { return string(a) }
