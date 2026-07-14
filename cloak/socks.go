package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

func listenSOCKS(addr string, p *peer) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}
		go handleSOCKS(c, p)
	}
}

func handleSOCKS(c net.Conn, p *peer) {
	defer c.Close()
	if err := socksHandshake(c); err != nil {
		log.Printf("socks handshake: %v", err)
		return
	}
	host, port, err := socksConnectReq(c)
	if err != nil {
		log.Printf("socks connect: %v", err)
		return
	}
	remote, err := dialThrough(p, host, port)
	if err != nil {
		_ = socksReply(c, 0x05) // connection refused
		log.Printf("tunnel open %s:%d: %v", host, port, err)
		return
	}
	defer remote.Close()
	if err := socksReply(c, 0x00); err != nil {
		return
	}
	pipe(c, remote)
}

func socksHandshake(c net.Conn) error {
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(c, hdr); err != nil {
		return err
	}
	if hdr[0] != 0x05 {
		return fmt.Errorf("not socks5")
	}
	methods := make([]byte, int(hdr[1]))
	if _, err := io.ReadFull(c, methods); err != nil {
		return err
	}
	// no auth
	_, err := c.Write([]byte{0x05, 0x00})
	return err
}

func socksConnectReq(c net.Conn) (string, uint16, error) {
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(c, hdr); err != nil {
		return "", 0, err
	}
	if hdr[0] != 0x05 || hdr[1] != 0x01 {
		return "", 0, fmt.Errorf("only CONNECT supported")
	}
	var host string
	switch hdr[3] {
	case 0x01: // IPv4
		ip := make([]byte, 4)
		if _, err := io.ReadFull(c, ip); err != nil {
			return "", 0, err
		}
		host = net.IP(ip).String()
	case 0x03: // domain
		var n [1]byte
		if _, err := io.ReadFull(c, n[:]); err != nil {
			return "", 0, err
		}
		name := make([]byte, n[0])
		if _, err := io.ReadFull(c, name); err != nil {
			return "", 0, err
		}
		host = string(name)
	case 0x04: // IPv6
		ip := make([]byte, 16)
		if _, err := io.ReadFull(c, ip); err != nil {
			return "", 0, err
		}
		host = net.IP(ip).String()
	default:
		return "", 0, fmt.Errorf("bad atyp %d", hdr[3])
	}
	var pb [2]byte
	if _, err := io.ReadFull(c, pb[:]); err != nil {
		return "", 0, err
	}
	port := binary.BigEndian.Uint16(pb[:])
	return host, port, nil
}

func socksReply(c net.Conn, status byte) error {
	// VER REP RSV ATYP BND.ADDR BND.PORT
	_, err := c.Write([]byte{0x05, status, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	return err
}

func pipe(a, b net.Conn) {
	done := make(chan struct{}, 2)
	cp := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}
	go cp(a, b)
	go cp(b, a)
	<-done
}
