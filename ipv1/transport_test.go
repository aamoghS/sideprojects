package ipv1

import (
	"net"
	"path/filepath"
	"testing"
	"time"
)

func TestUDPRoundTrip(t *testing.T) {
	dir := t.TempDir()
	reg, err := Open(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}

	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.LocalAddr().(*net.UDPAddr).Port
	endpoint := "127.0.0.1:" + itoa(port)

	if err := reg.Register("3", endpoint); err != nil {
		t.Fatal(err)
	}

	done := make(chan Packet, 1)
	go func() {
		buf := make([]byte, 4096)
		_ = ln.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, _, err := ln.ReadFromUDP(buf)
		if err != nil {
			return
		}
		pkt, err := Decode(buf[:n])
		if err != nil {
			return
		}
		done <- pkt
	}()

	raw, err := Encode(Packet{Src: "1", Dst: "3", Payload: []byte("hello")})
	if err != nil {
		t.Fatal(err)
	}
	ep, err := reg.Lookup("3")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("udp", ep)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write(raw); err != nil {
		t.Fatal(err)
	}

	select {
	case pkt := <-done:
		if pkt.Src != "1" || pkt.Dst != "3" || string(pkt.Payload) != "hello" {
			t.Fatalf("got %+v", pkt)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for packet")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
