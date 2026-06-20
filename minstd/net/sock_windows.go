//go:build windows

package net

import (
	"minstd/errors"
	"minstd/sync"
	"syscall"
	"unsafe"
)

var (
	modws2_32 = syscall.NewLazyDLL("ws2_32.dll")
	procSend  = modws2_32.NewProc("send")
	procRecv  = modws2_32.NewProc("recv")
	procAccept = modws2_32.NewProc("accept")
)

var (
	wsaMu    sync.Mutex
	wsaReady bool
	wsaErr   error
)

func ensureWSA() error {
	wsaMu.Lock()
	defer wsaMu.Unlock()
	if wsaReady {
		return wsaErr
	}
	var data syscall.WSAData
	wsaErr = syscall.WSAStartup(uint32(0x0202), &data)
	wsaReady = true
	return wsaErr
}

func listenTCP(host string, port int) (Listener, error) {
	if err := ensureWSA(); err != nil {
		return nil, err
	}
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		return nil, err
	}

	ip, err := parseIPv4(host)
	if err != nil {
		socketClose(uintptr(fd))
		return nil, err
	}
	sa := &syscall.SockaddrInet4{Port: port, Addr: ip}
	if err := syscall.Bind(fd, sa); err != nil {
		socketClose(uintptr(fd))
		return nil, err
	}
	if err := syscall.Listen(fd, syscall.SOMAXCONN); err != nil {
		socketClose(uintptr(fd))
		return nil, err
	}

	la := &tcpAddr{host: host, port: port}
	if port == 0 {
		if bound, err := socketAddr(fd); err == nil {
			la = bound
		}
	}

	return &tcpListener{fd: uintptr(fd), addr: la}, nil
}

func dialTCP(host string, port int) (Conn, error) {
	if err := ensureWSA(); err != nil {
		return nil, err
	}
	if host == "" {
		host = "127.0.0.1"
	}
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		return nil, err
	}

	ip, err := parseIPv4(host)
	if err != nil {
		socketClose(uintptr(fd))
		return nil, err
	}
	sa := &syscall.SockaddrInet4{Port: port, Addr: ip}
	if err := syscall.Connect(fd, sa); err != nil {
		socketClose(uintptr(fd))
		return nil, err
	}
	return &socket{fd: uintptr(fd)}, nil
}

func socketAccept(fd uintptr) (uintptr, error) {
	var sa syscall.RawSockaddrAny
	saLen := int32(unsafe.Sizeof(sa))
	r, _, e1 := procAccept.Call(uintptr(fd), uintptr(unsafe.Pointer(&sa)), uintptr(unsafe.Pointer(&saLen)))
	if r == ^uintptr(0) {
		if e1 != nil {
			return 0, e1
		}
		return 0, ErrClosed
	}
	return r, nil
}

func socketRead(fd uintptr, b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	r, _, e1 := procRecv.Call(uintptr(fd), uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)), 0)
	if r == ^uintptr(0) {
		if e1 != nil && e1 != syscall.EINVAL {
			return 0, e1
		}
		if e1 == syscall.EINVAL {
			return 0, e1
		}
		return 0, syscall.EINVAL
	}
	return int(r), nil
}

func socketWrite(fd uintptr, b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	r, _, e1 := procSend.Call(uintptr(fd), uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)), 0)
	if r == ^uintptr(0) {
		if e1 != nil {
			return 0, e1
		}
		return 0, syscall.EINVAL
	}
	return int(r), nil
}

func socketClose(fd uintptr) error {
	return syscall.Closesocket(syscall.Handle(fd))
}

func socketAddr(fd syscall.Handle) (*tcpAddr, error) {
	sa, err := syscall.Getsockname(fd)
	if err != nil {
		return nil, err
	}
	inet4, ok := sa.(*syscall.SockaddrInet4)
	if !ok {
		return nil, errors.New("unexpected sockaddr")
	}
	host := ipv4String(inet4.Addr)
	return &tcpAddr{host: host, port: inet4.Port}, nil
}

func ipv4String(ip [4]byte) string {
	return strconvItoa(int(ip[0])) + "." + strconvItoa(int(ip[1])) + "." + strconvItoa(int(ip[2])) + "." + strconvItoa(int(ip[3]))
}

func strconvItoa(n int) string {
	// keep net/ free of import cycles with strconv by inlining tiny helper
	if n == 0 {
		return "0"
	}
	var buf [3]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
