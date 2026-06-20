package net

import (
	"minstd/errors"
	"minstd/strconv"
	"minstd/strings"
)

type tcpAddr struct {
	host string
	port int
}

func (a *tcpAddr) String() string {
	host := a.host
	if host == "" {
		host = "0.0.0.0"
	}
	return host + ":" + strconv.Itoa(a.port)
}

func parseAddr(address string) (host string, port int, err error) {
	if address == "" {
		return "", 0, errors.New("empty address")
	}
	if address[0] == ':' {
		port, err = strconv.Atoi(address[1:])
		if err != nil {
			return "", 0, errors.New("invalid port")
		}
		return "", port, nil
	}
	h, p, ok := strings.Cut(address, ":")
	if !ok {
		return "", 0, errors.New("address must be host:port")
	}
	port, err = strconv.Atoi(p)
	if err != nil || port < 0 || port > 65535 {
		return "", 0, errors.New("invalid port")
	}
	return h, port, nil
}

func parseIPv4(host string) ([4]byte, error) {
	if host == "" {
		return [4]byte{}, nil
	}
	parts := strings.SplitN(host, ".", 4)
	if len(parts) != 4 {
		return [4]byte{}, errors.New("invalid ipv4")
	}
	var out [4]byte
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || n > 255 {
			return [4]byte{}, errors.New("invalid ipv4")
		}
		out[i] = byte(n)
	}
	return out, nil
}

func htons(port int) uint16 {
	p := uint16(port)
	return (p<<8)&0xff00 | (p>>8)&0xff
}

func ntohs(port uint16) int {
	return int((port<<8)&0xff00 | (port>>8)&0xff)
}
