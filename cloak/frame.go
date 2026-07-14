package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"golang.org/x/crypto/nacl/secretbox"
)

const frameVersion byte = 1

var (
	errShortPacket  = errors.New("cloak: short packet")
	errBadVersion   = errors.New("cloak: bad version")
	errDecryptFailed = errors.New("cloak: decrypt failed")
	errShortMsg     = errors.New("cloak: short message")
)

// Wire packet: version (1) + nonce (24) + secretbox(ciphertext).
func seal(key *[32]byte, plain []byte) ([]byte, error) {
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	out := make([]byte, 1+24, 1+24+len(plain)+secretbox.Overhead)
	out[0] = frameVersion
	copy(out[1:25], nonce[:])
	return secretbox.Seal(out, plain, &nonce, key), nil
}

func open(key *[32]byte, packet []byte) ([]byte, error) {
	if len(packet) < 1+24+secretbox.Overhead {
		return nil, errShortPacket
	}
	if packet[0] != frameVersion {
		return nil, errBadVersion
	}
	var nonce [24]byte
	copy(nonce[:], packet[1:25])
	plain, ok := secretbox.Open(nil, packet[25:], &nonce, key)
	if !ok {
		return nil, errDecryptFailed
	}
	return plain, nil
}

// Inner messages: stream types for optional SOCKS mode, packet/hello for L3 VPN.
const (
	msgOpen    byte = 1
	msgOpenAck byte = 2
	msgOpenErr byte = 3
	msgData    byte = 4
	msgAck     byte = 5
	msgClose   byte = 6
	msgPacket  byte = 10 // body = raw IPv4/IPv6 packet
	msgHello   byte = 11 // body = 4-byte tunnel IPv4
)

type message struct {
	Type     byte
	StreamID uint32
	Seq      uint32
	Body     []byte
}

func encodeMsg(m message) []byte {
	out := make([]byte, 1+4+4+len(m.Body))
	out[0] = m.Type
	binary.BigEndian.PutUint32(out[1:5], m.StreamID)
	binary.BigEndian.PutUint32(out[5:9], m.Seq)
	copy(out[9:], m.Body)
	return out
}

func decodeMsg(b []byte) (message, error) {
	if len(b) < 9 {
		return message{}, errShortMsg
	}
	return message{
		Type:     b[0],
		StreamID: binary.BigEndian.Uint32(b[1:5]),
		Seq:      binary.BigEndian.Uint32(b[5:9]),
		Body:     append([]byte(nil), b[9:]...),
	}, nil
}

// parseKey accepts 64-char hex (32 raw bytes) or any passphrase (sha256).
func parseKey(s string) (*[32]byte, error) {
	if s == "" {
		return nil, fmt.Errorf("cloak: empty key")
	}
	var key [32]byte
	if len(s) == 64 {
		raw, err := hex.DecodeString(s)
		if err == nil && len(raw) == 32 {
			copy(key[:], raw)
			return &key, nil
		}
	}
	sum := sha256.Sum256([]byte(s))
	copy(key[:], sum[:])
	return &key, nil
}
