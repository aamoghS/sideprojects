package io

import stdio "io"

var (
	EOF             = stdio.EOF
	ErrUnexpectedEOF = stdio.ErrUnexpectedEOF
)

type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}

func ReadFull(r Reader, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		got, err := r.Read(buf[n:])
		n += got
		if err != nil {
			return n, err
		}
		if got == 0 {
			return n, ErrUnexpectedEOF
		}
	}
	return n, nil
}

func ReadAll(r Reader) ([]byte, error) {
	buf := make([]byte, 4096)
	var out []byte
	for {
		n, err := r.Read(buf)
		if n > 0 {
			out = append(out, buf[:n]...)
		}
		if err != nil {
			if err == EOF {
				return out, nil
			}
			return out, err
		}
	}
}
