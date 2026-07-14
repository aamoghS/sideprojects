package main

import (
	"bytes"
	"testing"
)

func TestSealOpenRoundTrip(t *testing.T) {
	key, err := parseKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("hello through the cloak")
	pkt, err := seal(key, plain)
	if err != nil {
		t.Fatal(err)
	}
	if pkt[0] != frameVersion {
		t.Fatalf("version: got %d", pkt[0])
	}
	got, err := open(key, pkt)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("plain mismatch: %q vs %q", got, plain)
	}
}

func TestOpenBadKey(t *testing.T) {
	k1, _ := parseKey("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	k2, _ := parseKey("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	pkt, err := seal(k1, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := open(k2, pkt); err != errDecryptFailed {
		t.Fatalf("want decrypt failed, got %v", err)
	}
}

func TestOpenShort(t *testing.T) {
	key, _ := parseKey("passphrase-works-too")
	if _, err := open(key, []byte{1, 2, 3}); err != errShortPacket {
		t.Fatalf("want short packet, got %v", err)
	}
}

func TestOpenBadVersion(t *testing.T) {
	key, _ := parseKey("passphrase-works-too")
	pkt, err := seal(key, []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	pkt[0] = 99
	if _, err := open(key, pkt); err != errBadVersion {
		t.Fatalf("want bad version, got %v", err)
	}
}

func TestMsgEncodeDecode(t *testing.T) {
	m := message{Type: msgData, StreamID: 42, Seq: 7, Body: []byte("payload")}
	raw := encodeMsg(m)
	got, err := decodeMsg(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != m.Type || got.StreamID != m.StreamID || got.Seq != m.Seq || !bytes.Equal(got.Body, m.Body) {
		t.Fatalf("mismatch: %+v vs %+v", got, m)
	}
}

func TestParseKeyHexAndPassphrase(t *testing.T) {
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	k1, err := parseKey(hexKey)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := parseKey(hexKey)
	if err != nil {
		t.Fatal(err)
	}
	if *k1 != *k2 {
		t.Fatal("hex key not stable")
	}
	p1, err := parseKey("my-lab-psk")
	if err != nil {
		t.Fatal(err)
	}
	p2, err := parseKey("my-lab-psk")
	if err != nil {
		t.Fatal(err)
	}
	if *p1 != *p2 {
		t.Fatal("passphrase key not stable")
	}
	if *p1 == *k1 {
		t.Fatal("passphrase collided with hex key")
	}
}

func TestDecodeMsgShort(t *testing.T) {
	if _, err := decodeMsg([]byte{1, 2, 3}); err != errShortMsg {
		t.Fatalf("want short msg, got %v", err)
	}
}

func TestPacketMsgRoundTrip(t *testing.T) {
	ip := []byte{
		0x45, 0x00, 0x00, 0x14, 0x00, 0x00, 0x00, 0x00, 0x40, 0x01, 0x00, 0x00,
		10, 8, 0, 2, 10, 8, 0, 1,
	}
	raw := encodeMsg(message{Type: msgPacket, Body: ip})
	got, err := decodeMsg(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != msgPacket || !bytes.Equal(got.Body, ip) {
		t.Fatalf("packet msg mismatch: %+v", got)
	}
	dst, ok := ipv4Dst(ip)
	if !ok || dst != [4]byte{10, 8, 0, 1} {
		t.Fatalf("dst: %v %v", dst, ok)
	}
	src, ok := ipv4Src(ip)
	if !ok || src != [4]byte{10, 8, 0, 2} {
		t.Fatalf("src: %v %v", src, ok)
	}
}

func TestSubnetGateway(t *testing.T) {
	gw, err := subnetGateway("10.8.0.2/24")
	if err != nil {
		t.Fatal(err)
	}
	if gw.String() != "10.8.0.1" {
		t.Fatalf("gateway: %s", gw)
	}
}
