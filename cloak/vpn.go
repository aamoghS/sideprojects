package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"golang.zx2c4.com/wireguard/tun"
)

const (
	defaultMTU      = 1420
	helloInterval   = 15 * time.Second
	tunReadBufSize  = 2048
)

type vpnOpts struct {
	key     *[32]byte
	tunName string
	cidr    string
	routes  []string // extra CIDRs routed via tunnel gateway
}

func openTun(opts vpnOpts) (tun.Device, error) {
	dev, err := tun.CreateTUN(opts.tunName, defaultMTU)
	if err != nil {
		return nil, fmt.Errorf("create TUN %q: %w (need admin/sudo; on Windows put wintun.dll next to cloak.exe)", opts.tunName, err)
	}
	if err := configureDevice(dev, opts.cidr); err != nil {
		dev.Close()
		return nil, err
	}
	name, _ := dev.Name()
	log.Printf("tun %s up %s", name, opts.cidr)

	gw, err := subnetGateway(opts.cidr)
	if err != nil {
		dev.Close()
		return nil, err
	}
	for _, r := range opts.routes {
		if err := addRouteVia(dev, r, gw.String()); err != nil {
			dev.Close()
			return nil, fmt.Errorf("route %s: %w", r, err)
		}
		log.Printf("route %s via %s", r, gw)
	}
	return dev, nil
}

func runVPNServer(listen string, opts vpnOpts) error {
	dev, err := openTun(opts)
	if err != nil {
		return err
	}
	defer dev.Close()

	addr, err := net.ResolveUDPAddr("udp", listen)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	hub := &vpnHub{conn: conn, key: opts.key, dev: dev, byIP: make(map[[4]byte]*net.UDPAddr)}
	log.Printf("cloak vpn server on %s", conn.LocalAddr())

	errCh := make(chan error, 2)
	go func() { errCh <- hub.tunToUDP() }()
	go func() { errCh <- hub.udpToTun() }()
	return <-errCh
}

func runVPNClient(serverAddr string, opts vpnOpts) error {
	dev, err := openTun(opts)
	if err != nil {
		return err
	}
	defer dev.Close()

	raddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return err
	}
	defer conn.Close()

	localIP, _, err := net.ParseCIDR(opts.cidr)
	if err != nil {
		return err
	}
	v4 := localIP.To4()
	if v4 == nil {
		return fmt.Errorf("cloak: need IPv4 -addr")
	}

	c := &vpnClient{
		conn:   conn,
		remote: raddr,
		key:    opts.key,
		dev:    dev,
		local:  [4]byte{v4[0], v4[1], v4[2], v4[3]},
	}
	log.Printf("cloak vpn client -> %s", serverAddr)
	if err := c.sendHello(); err != nil {
		return err
	}

	errCh := make(chan error, 3)
	go func() { errCh <- c.helloLoop() }()
	go func() { errCh <- c.tunToUDP() }()
	go func() { errCh <- c.udpToTun() }()
	return <-errCh
}

type vpnHub struct {
	conn *net.UDPConn
	key  *[32]byte
	dev  tun.Device

	mu   sync.Mutex
	byIP map[[4]byte]*net.UDPAddr
	last *net.UDPAddr
}

func (h *vpnHub) remember(ip [4]byte, addr *net.UDPAddr) {
	h.mu.Lock()
	h.byIP[ip] = cloneUDPAddr(addr)
	h.last = cloneUDPAddr(addr)
	h.mu.Unlock()
}

func (h *vpnHub) lookup(ip [4]byte) *net.UDPAddr {
	h.mu.Lock()
	defer h.mu.Unlock()
	if a, ok := h.byIP[ip]; ok {
		return a
	}
	return h.last
}

func (h *vpnHub) send(remote *net.UDPAddr, m message) error {
	pkt, err := seal(h.key, encodeMsg(m))
	if err != nil {
		return err
	}
	_, err = h.conn.WriteToUDP(pkt, remote)
	return err
}

func (h *vpnHub) tunToUDP() error {
	batch := h.dev.BatchSize()
	bufs := make([][]byte, batch)
	sizes := make([]int, batch)
	for i := range bufs {
		bufs[i] = make([]byte, tunReadBufSize)
	}
	for {
		n, err := h.dev.Read(bufs, sizes, 0)
		if err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			pkt := bufs[i][:sizes[i]]
			dst, ok := ipv4Dst(pkt)
			if !ok {
				continue
			}
			remote := h.lookup(dst)
			if remote == nil {
				continue
			}
			_ = h.send(remote, message{Type: msgPacket, Body: append([]byte(nil), pkt...)})
		}
	}
}

func (h *vpnHub) udpToTun() error {
	buf := make([]byte, maxPacket)
	out := make([][]byte, 1)
	for {
		n, addr, err := h.conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		plain, err := open(h.key, buf[:n])
		if err != nil {
			continue
		}
		msg, err := decodeMsg(plain)
		if err != nil {
			continue
		}
		switch msg.Type {
		case msgHello:
			if len(msg.Body) != 4 {
				continue
			}
			var ip [4]byte
			copy(ip[:], msg.Body)
			h.remember(ip, addr)
			log.Printf("peer %s tun %d.%d.%d.%d", addr, ip[0], ip[1], ip[2], ip[3])
		case msgPacket:
			if len(msg.Body) == 0 {
				continue
			}
			if src, ok := ipv4Src(msg.Body); ok {
				h.remember(src, addr)
			}
			out[0] = msg.Body
			_, _ = h.dev.Write(out, 0)
		}
	}
}

type vpnClient struct {
	conn   *net.UDPConn
	remote *net.UDPAddr
	key    *[32]byte
	dev    tun.Device
	local  [4]byte
}

func (c *vpnClient) send(m message) error {
	pkt, err := seal(c.key, encodeMsg(m))
	if err != nil {
		return err
	}
	_, err = c.conn.WriteToUDP(pkt, c.remote)
	return err
}

func (c *vpnClient) sendHello() error {
	return c.send(message{Type: msgHello, Body: c.local[:]})
}

func (c *vpnClient) helloLoop() error {
	t := time.NewTicker(helloInterval)
	defer t.Stop()
	for range t.C {
		if err := c.sendHello(); err != nil {
			return err
		}
	}
	return nil
}

func (c *vpnClient) tunToUDP() error {
	batch := c.dev.BatchSize()
	bufs := make([][]byte, batch)
	sizes := make([]int, batch)
	for i := range bufs {
		bufs[i] = make([]byte, tunReadBufSize)
	}
	for {
		n, err := c.dev.Read(bufs, sizes, 0)
		if err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			pkt := append([]byte(nil), bufs[i][:sizes[i]]...)
			if err := c.send(message{Type: msgPacket, Body: pkt}); err != nil {
				return err
			}
		}
	}
}

func (c *vpnClient) udpToTun() error {
	buf := make([]byte, maxPacket)
	out := make([][]byte, 1)
	for {
		n, addr, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		if addr.String() != c.remote.String() {
			continue
		}
		plain, err := open(c.key, buf[:n])
		if err != nil {
			continue
		}
		msg, err := decodeMsg(plain)
		if err != nil || msg.Type != msgPacket || len(msg.Body) == 0 {
			continue
		}
		out[0] = msg.Body
		_, _ = c.dev.Write(out, 0)
	}
}

func ipv4Dst(pkt []byte) ([4]byte, bool) {
	var z [4]byte
	if len(pkt) < 20 || pkt[0]>>4 != 4 {
		return z, false
	}
	copy(z[:], pkt[16:20])
	return z, true
}

func ipv4Src(pkt []byte) ([4]byte, bool) {
	var z [4]byte
	if len(pkt) < 20 || pkt[0]>>4 != 4 {
		return z, false
	}
	copy(z[:], pkt[12:16])
	return z, true
}

func subnetGateway(cidr string) (net.IP, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	ip := ipnet.IP.To4()
	if ip == nil {
		return nil, fmt.Errorf("cloak: IPv4 cidr required")
	}
	gw := make(net.IP, 4)
	copy(gw, ip)
	for i := 3; i >= 0; i-- {
		gw[i]++
		if gw[i] != 0 {
			break
		}
	}
	return gw, nil
}

func cloneUDPAddr(a *net.UDPAddr) *net.UDPAddr {
	if a == nil {
		return nil
	}
	ip := make(net.IP, len(a.IP))
	copy(ip, a.IP)
	return &net.UDPAddr{IP: ip, Port: a.Port, Zone: a.Zone}
}
