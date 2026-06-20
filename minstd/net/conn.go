package net

type socket struct {
	fd uintptr
}

func (s *socket) Read(b []byte) (int, error)  { return socketRead(s.fd, b) }
func (s *socket) Write(b []byte) (int, error) { return socketWrite(s.fd, b) }
func (s *socket) Close() error                { return socketClose(s.fd) }

type tcpListener struct {
	fd   uintptr
	addr *tcpAddr
}

func (l *tcpListener) Accept() (Conn, error) {
	nfd, err := socketAccept(l.fd)
	if err != nil {
		return nil, err
	}
	return &socket{fd: nfd}, nil
}

func (l *tcpListener) Close() error {
	return socketClose(l.fd)
}

func (l *tcpListener) Addr() Addr {
	return l.addr
}
