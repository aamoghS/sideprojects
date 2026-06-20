package http

import (
	"minstd/net"
	"minstd/strconv"
)

type ResponseWriter interface {
	Header() map[string]string
	WriteHeader(code int)
	Write(b []byte) (int, error)
}

type responseWriter struct {
	conn        net.Conn
	status      int
	wroteHeader bool
	header      map[string]string
}

func newResponseWriter(conn net.Conn) *responseWriter {
	return &responseWriter{
		conn:   conn,
		header: make(map[string]string),
	}
}

func (w *responseWriter) Header() map[string]string {
	return w.header
}

func (w *responseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.status = code
	w.wroteHeader = true
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(StatusOK)
	}
	if _, ok := w.header["Content-Type"]; !ok {
		w.header["Content-Type"] = "text/plain; charset=utf-8"
	}
	w.header["Content-Length"] = strconv.Itoa(len(b))
	w.header["Connection"] = "close"

	out := appendStatusLine(nil, w.status)
	for k, v := range w.header {
		out = appendHeaderLine(out, k, v)
	}
	out = append(out, '\r', '\n')
	out = append(out, b...)

	_, err := w.conn.Write(out)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func appendStatusLine(out []byte, code int) []byte {
	out = append(out, "HTTP/1.1 "...)
	out = append(out, strconv.Itoa(code)...)
	out = append(out, ' ')
	out = append(out, statusText(code)...)
	return append(out, '\r', '\n')
}

func appendHeaderLine(out []byte, key, val string) []byte {
	out = append(out, key...)
	out = append(out, ':', ' ')
	out = append(out, val...)
	return append(out, '\r', '\n')
}
