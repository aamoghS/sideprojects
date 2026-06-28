//go:build !windows

package net

import (
	"github.com/aamoghS/sideprojects/minstd/errors"
	"syscall"
)

func listenTCP(host string, port int) (Listener, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}

	ip, err := parseIPv4(host)
	if err != nil {
		socketClose(fd)
		return nil, err
	}
	sa := &syscall.SockaddrInet4{Port: port, Addr: ip}
	if err := syscall.Bind(fd, sa); err != nil {
		socketClose(fd)
		return nil, err
	}
	if err := syscall.Listen(fd, syscall.SOMAXCONN); err != nil {
		socketClose(fd)
		return nil, err
	}

	la := &tcpAddr{host: host, port: port}
	if port == 0 {
		if bound, err := socketAddr(fd); err == nil {
			la = bound
		}
	}
	return &tcpListener{fd: uintptr(fd), addr: la}, nil
}

func dialTCP(host string, port int) (Conn, error) {
	if host == "" {
		host = "127.0.0.1"
	}
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}

	ip, err := parseIPv4(host)
	if err != nil {
		socketClose(fd)
		return nil, err
	}
	sa := &syscall.SockaddrInet4{Port: port, Addr: ip}
	if err := syscall.Connect(fd, sa); err != nil {
		socketClose(fd)
		return nil, err
	}
	return &socket{fd: uintptr(fd)}, nil
}

func socketAccept(fd uintptr) (uintptr, error) {
	nfd, _, err := syscall.Accept(int(fd))
	if err != nil {
		return 0, err
	}
	return uintptr(nfd), nil
}

func socketRead(fd uintptr, b []byte) (int, error) {
	return syscall.Read(int(fd), b)
}

func socketWrite(fd uintptr, b []byte) (int, error) {
	return syscall.Write(int(fd), b)
}

func socketClose(fd uintptr) error {
	return syscall.Close(int(fd))
}

func socketAddr(fd int) (*tcpAddr, error) {
	sa, err := syscall.Getsockname(fd)
	if err != nil {
		return nil, err
	}
	inet4, ok := sa.(*syscall.SockaddrInet4)
	if !ok {
		return nil, errors.New("unexpected sockaddr")
	}
	return &tcpAddr{host: ipv4String(inet4.Addr), port: inet4.Port}, nil
}

func ipv4String(ip [4]byte) string {
	return byteString(ip[0]) + "." + byteString(ip[1]) + "." + byteString(ip[2]) + "." + byteString(ip[3])
}

func byteString(b byte) string {
	if b == 0 {
		return "0"
	}
	var digits [3]byte
	n := int(b)
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}
