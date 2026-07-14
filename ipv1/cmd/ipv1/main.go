package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aamoghS/sideprojects/ipv1"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "register":
		cmdRegister(os.Args[2:])
	case "send":
		cmdSend(os.Args[2:])
	case "listen":
		cmdListen(os.Args[2:])
	case "allocate":
		cmdAllocate(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `ipv1 - flat-address UDP toy protocol

Usage:
  ipv1 register <addr> <host:port> [--registry path]
  ipv1 allocate <host:port> [--registry path]
  ipv1 send <src> <dst> <message> [--registry path]
  ipv1 listen <addr> [--port N] [--registry path]

`)
}

func registryFlag(args []string) (positional []string, regPath string) {
	fs := flag.NewFlagSet("ipv1", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("registry", "", "registry file (default .ipv1/registry.json)")
	flagArgs, pos := splitFlags(args)
	_ = fs.Parse(flagArgs)
	return pos, *path
}

func splitFlags(args []string) (flags []string, positional []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return flags, append(positional, args[i+1:]...)
		}
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") && !strings.Contains(arg, "=") {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		positional = append(positional, arg)
	}
	return flags, positional
}

func openRegistry(path string) *ipv1.Registry {
	r, err := ipv1.Open(path)
	if err != nil {
		fail(err)
	}
	return r
}

func cmdRegister(args []string) {
	pos, path := registryFlag(args)
	if len(pos) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ipv1 register <addr> <host:port>")
		os.Exit(1)
	}
	r := openRegistry(path)
	if err := r.Register(pos[0], pos[1]); err != nil {
		fail(err)
	}
	fmt.Printf("registered %s -> %s\n", pos[0], pos[1])
}

func cmdAllocate(args []string) {
	pos, path := registryFlag(args)
	if len(pos) < 1 {
		fmt.Fprintln(os.Stderr, "usage: ipv1 allocate <host:port>")
		os.Exit(1)
	}
	r := openRegistry(path)
	addr, err := r.Allocate(pos[0])
	if err != nil {
		fail(err)
	}
	fmt.Println(addr)
}

func cmdSend(args []string) {
	pos, path := registryFlag(args)
	if len(pos) < 3 {
		fmt.Fprintln(os.Stderr, "usage: ipv1 send <src> <dst> <message>")
		os.Exit(1)
	}
	src, dst := pos[0], pos[1]
	msg := strings.Join(pos[2:], " ")

	r := openRegistry(path)
	ep, err := r.Lookup(dst)
	if err != nil {
		fail(err)
	}

	raw, err := ipv1.Encode(ipv1.Packet{Src: src, Dst: dst, Payload: []byte(msg)})
	if err != nil {
		fail(err)
	}

	conn, err := net.Dial("udp", ep)
	if err != nil {
		fail(err)
	}
	defer conn.Close()

	if _, err := conn.Write(raw); err != nil {
		fail(err)
	}
	fmt.Printf("sent %q %s -> %s (%s)\n", msg, src, dst, ep)
}

func cmdListen(args []string) {
	fs := flag.NewFlagSet("listen", flag.ExitOnError)
	port := fs.Int("port", 9876, "UDP listen port")
	regPath := fs.String("registry", "", "registry file")
	flagArgs, positional := splitFlags(args)
	_ = fs.Parse(flagArgs)

	if len(positional) < 1 {
		fmt.Fprintln(os.Stderr, "usage: ipv1 listen <addr> [--port N]")
		os.Exit(1)
	}
	addr := positional[0]

	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", *port))
	if err != nil {
		fail(err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fail(err)
	}
	defer conn.Close()

	endpoint, err := localEndpoint(conn.LocalAddr(), *port)
	if err != nil {
		fail(err)
	}

	r := openRegistry(*regPath)
	if err := r.Register(addr, endpoint); err != nil {
		fail(err)
	}
	fmt.Fprintf(os.Stderr, "listening as %s on %s\n", addr, endpoint)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	buf := make([]byte, 64*1024)
	for {
		if err := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			fail(err)
		}
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			if ctx.Err() != nil {
				return
			}
			fail(err)
		}
		pkt, err := ipv1.Decode(buf[:n])
		if err != nil {
			fmt.Fprintf(os.Stderr, "bad packet: %v\n", err)
			continue
		}
		if pkt.Dst != addr {
			continue
		}
		fmt.Printf("%s -> %s: %s\n", pkt.Src, pkt.Dst, string(pkt.Payload))
	}
}

func localEndpoint(addr net.Addr, port int) (string, error) {
	host := "127.0.0.1"
	if udp, ok := addr.(*net.UDPAddr); ok && udp.IP != nil && !udp.IP.IsUnspecified() {
		host = udp.IP.String()
	}
	return fmt.Sprintf("%s:%d", host, port), nil
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
