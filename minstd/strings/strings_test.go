package strings

import "testing"

func TestSplitN(t *testing.T) {
	got := SplitN("GET / HTTP/1.1", " ", 3)
	if len(got) != 3 || got[0] != "GET" || got[1] != "/" || got[2] != "HTTP/1.1" {
		t.Fatalf("SplitN = %#v", got)
	}
}

func TestCut(t *testing.T) {
	before, after, ok := Cut("Host: localhost", ":")
	if !ok || before != "Host" || TrimSpace(after) != "localhost" {
		t.Fatalf("Cut = %q %q %v", before, after, ok)
	}
}
