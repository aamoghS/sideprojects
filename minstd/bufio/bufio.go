package bufio

import "minstd/io"

type Reader struct {
	src io.Reader
	buf []byte
	off int
	end int
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		src: r,
		buf: make([]byte, 4096),
	}
}

func (r *Reader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.off >= r.end {
		n, err := r.src.Read(r.buf)
		r.off = 0
		r.end = n
		if n == 0 {
			return 0, err
		}
	}
	n := copy(p, r.buf[r.off:r.end])
	r.off += n
	return n, nil
}

func (r *Reader) ReadString(delim byte) (string, error) {
	var out []byte
	for {
		if r.off >= r.end {
			n, err := r.src.Read(r.buf)
			r.off = 0
			r.end = n
			if n == 0 {
				if len(out) == 0 {
					if err != nil {
						return "", err
					}
					return "", io.EOF
				}
				if err != nil {
					return string(out), err
				}
				return string(out), io.EOF
			}
		}
		for r.off < r.end {
			c := r.buf[r.off]
			r.off++
			out = append(out, c)
			if c == delim {
				return string(out), nil
			}
		}
	}
}
