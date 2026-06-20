package net

import (
	"testing"
)

func TestRoundTrip(t *testing.T) {
	ln, err := Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		defer c.Close()
		buf := make([]byte, 64)
		n, err := c.Read(buf)
		if err != nil {
			t.Error(err)
			return
		}
		if _, err := c.Write(buf[:n]); err != nil {
			t.Error(err)
		}
	}()

	client, err := Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	msg := []byte("ping")
	if _, err := client.Write(msg); err != nil {
		t.Fatal(err)
	}
	out := make([]byte, len(msg))
	if _, err := client.Read(out); err != nil {
		t.Fatal(err)
	}
	if string(out) != "ping" {
		t.Fatalf("got %q", out)
	}
}
