package ipv1

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const Version byte = 1

var (
	ErrBadVersion = errors.New("ipv1: bad version")
	ErrAddrTooLong = errors.New("ipv1: address too long")
	ErrTruncated   = errors.New("ipv1: truncated packet")
)

const maxAddrLen = 64

type Packet struct {
	Src     string
	Dst     string
	Payload []byte
}

func Encode(p Packet) ([]byte, error) {
	if err := validateAddr(p.Src); err != nil {
		return nil, fmt.Errorf("src: %w", err)
	}
	if err := validateAddr(p.Dst); err != nil {
		return nil, fmt.Errorf("dst: %w", err)
	}

	src := []byte(p.Src)
	dst := []byte(p.Dst)
	if len(src) > maxAddrLen || len(dst) > maxAddrLen {
		return nil, ErrAddrTooLong
	}

	out := make([]byte, 1+2+len(src)+2+len(dst)+len(p.Payload))
	out[0] = Version
	n := 1
	n += putLenPrefixed(out[n:], src)
	n += putLenPrefixed(out[n:], dst)
	copy(out[n:], p.Payload)
	return out, nil
}

func Decode(b []byte) (Packet, error) {
	if len(b) < 5 {
		return Packet{}, ErrTruncated
	}
	if b[0] != Version {
		return Packet{}, ErrBadVersion
	}

	off := 1
	src, n, err := getLenPrefixed(b[off:])
	if err != nil {
		return Packet{}, err
	}
	off += n

	dst, n, err := getLenPrefixed(b[off:])
	if err != nil {
		return Packet{}, err
	}
	off += n

	if err := validateAddr(src); err != nil {
		return Packet{}, fmt.Errorf("src: %w", err)
	}
	if err := validateAddr(dst); err != nil {
		return Packet{}, fmt.Errorf("dst: %w", err)
	}

	payload := append([]byte(nil), b[off:]...)
	return Packet{Src: src, Dst: dst, Payload: payload}, nil
}

func validateAddr(s string) error {
	if s == "" {
		return errors.New("empty address")
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '-':
		default:
			return fmt.Errorf("invalid address %q", s)
		}
	}
	return nil
}

func putLenPrefixed(dst, src []byte) int {
	binary.BigEndian.PutUint16(dst[:2], uint16(len(src)))
	copy(dst[2:], src)
	return 2 + len(src)
}

func getLenPrefixed(b []byte) (string, int, error) {
	if len(b) < 2 {
		return "", 0, ErrTruncated
	}
	n := int(binary.BigEndian.Uint16(b[:2]))
	if len(b) < 2+n {
		return "", 0, ErrTruncated
	}
	return string(b[2 : 2+n]), 2 + n, nil
}
