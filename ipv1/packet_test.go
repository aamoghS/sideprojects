package ipv1

import (
	"bytes"
	"errors"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	cases := []Packet{
		{Src: "1", Dst: "3", Payload: []byte("hello")},
		{Src: "42", Dst: "host-3", Payload: nil},
		{Src: "host-3", Dst: "1", Payload: []byte{0, 1, 2}},
	}
	for _, tc := range cases {
		raw, err := Encode(tc)
		if err != nil {
			t.Fatalf("encode %v: %v", tc, err)
		}
		if raw[0] != Version {
			t.Fatalf("version byte = %d", raw[0])
		}
		got, err := Decode(raw)
		if err != nil {
			t.Fatalf("decode %v: %v", tc, err)
		}
		if got.Src != tc.Src || got.Dst != tc.Dst || !bytes.Equal(got.Payload, tc.Payload) {
			t.Fatalf("round trip: got %+v want %+v", got, tc)
		}
	}
}

func TestDecodeErrors(t *testing.T) {
	if _, err := Decode([]byte{2, 0, 1, '1', 0, 1, '2'}); !errors.Is(err, ErrBadVersion) {
		t.Fatalf("bad version: %v", err)
	}
	if _, err := Decode([]byte{1, 0, 5}); !errors.Is(err, ErrTruncated) {
		t.Fatalf("truncated: %v", err)
	}
}

func TestValidateAddr(t *testing.T) {
	bad := []string{"", "1.2.3.4", "host_3", "a b"}
	for _, s := range bad {
		if err := validateAddr(s); err == nil {
			t.Fatalf("expected error for %q", s)
		}
	}
}
