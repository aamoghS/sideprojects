package http

import (
	"testing"

	"github.com/aamoghS/sideprojects/minstd/bufio"
	"github.com/aamoghS/sideprojects/minstd/io"
	"github.com/aamoghS/sideprojects/minstd/net"
	"github.com/aamoghS/sideprojects/minstd/strings"
)

func TestServerGET(t *testing.T) {
	mux := NewServeMux()
	mux.HandleFunc("/", func(w ResponseWriter, r *Request) {
		if r.Method != "GET" {
			t.Fatalf("method = %q", r.Method)
		}
		_, _ = w.Write([]byte("hello"))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		c, err := ln.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		srv := &Server{Handler: mux}
		srv.serveConn(c)
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	req := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatal(err)
	}

	br := bufio.NewReader(conn)
	status, err := br.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(status, "200") {
		t.Fatalf("status = %q", status)
	}

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(line) == "" {
			break
		}
	}

	body := make([]byte, 5)
	if _, err := io.ReadFull(br, body); err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello" {
		t.Fatalf("body = %q", body)
	}
}

func TestErrorResponse(t *testing.T) {
	p := &pipeConn{}
	w := newResponseWriter(p)
	Error(w, "boom", StatusInternalServerError)
	if !strings.Contains(string(p.wrote), "500") {
		t.Fatalf("response = %q", p.wrote)
	}
}

type pipeConn struct {
	wrote []byte
}

func (p *pipeConn) Read(b []byte) (int, error)  { return 0, io.EOF }
func (p *pipeConn) Write(b []byte) (int, error) { p.wrote = append(p.wrote, b...); return len(b), nil }
func (p *pipeConn) Close() error                { return nil }
