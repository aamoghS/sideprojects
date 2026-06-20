package http

import (
	"minstd/bufio"
	"minstd/io"
	"minstd/net"
	"minstd/strings"
)

type Request struct {
	Method string
	Path   string
	Proto  string
	Header map[string]string
}

func readRequest(conn net.Conn) (*Request, error) {
	br := bufio.NewReader(conn)
	line, err := br.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		return nil, io.ErrUnexpectedEOF
	}

	req := &Request{
		Method: parts[0],
		Path:   parts[1],
		Header: make(map[string]string),
	}
	if len(parts) == 3 {
		req.Proto = parts[2]
	}

	for {
		hline, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		hline = strings.TrimSpace(hline)
		if hline == "" {
			break
		}
		key, val, ok := strings.Cut(hline, ":")
		if !ok {
			continue
		}
		req.Header[strings.TrimSpace(key)] = strings.TrimSpace(val)
	}
	return req, nil
}
