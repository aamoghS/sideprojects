package net

import "minstd/errors"

var ErrClosed = errors.New("use of closed network connection")

type Conn interface {
	Read(b []byte) (int, error)
	Write(b []byte) (int, error)
	Close() error
}

type Addr interface {
	String() string
}

type Listener interface {
	Accept() (Conn, error)
	Close() error
	Addr() Addr
}

func Listen(network, address string) (Listener, error) {
	if network != "" && network != "tcp" {
		return nil, errors.New("unsupported network " + network)
	}
	host, port, err := parseAddr(address)
	if err != nil {
		return nil, err
	}
	return listenTCP(host, port)
}

func Dial(network, address string) (Conn, error) {
	if network != "" && network != "tcp" {
		return nil, errors.New("unsupported network " + network)
	}
	host, port, err := parseAddr(address)
	if err != nil {
		return nil, err
	}
	return dialTCP(host, port)
}
