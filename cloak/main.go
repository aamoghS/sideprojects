package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "server":
		fs := flag.NewFlagSet("server", flag.ExitOnError)
		listen := fs.String("listen", ":51820", "UDP listen address")
		key := fs.String("key", envOr("CLOAK_KEY", ""), "shared key (hex32 or passphrase)")
		tunName := fs.String("tun", "cloak0", "TUN interface name")
		addr := fs.String("addr", "10.8.0.1/24", "tunnel address CIDR")
		socks := fs.Bool("socks", false, "SOCKS5 exit mode instead of TUN")
		route := fs.String("route", "", "extra CIDRs to route via tunnel gateway (comma-separated)")
		_ = fs.Parse(os.Args[2:])
		if *key == "" {
			log.Fatal("need -key or CLOAK_KEY")
		}
		if *socks {
			if err := runSOCKSServer(*listen, *key); err != nil {
				log.Fatal(err)
			}
			return
		}
		k, err := parseKey(*key)
		if err != nil {
			log.Fatal(err)
		}
		if err := runVPNServer(*listen, vpnOpts{
			key:     k,
			tunName: *tunName,
			cidr:    *addr,
			routes:  splitCSV(*route),
		}); err != nil {
			log.Fatal(err)
		}
	case "client":
		fs := flag.NewFlagSet("client", flag.ExitOnError)
		server := fs.String("server", "", "server host:port")
		key := fs.String("key", envOr("CLOAK_KEY", ""), "shared key (hex32 or passphrase)")
		tunName := fs.String("tun", "cloak0", "TUN interface name")
		addr := fs.String("addr", "10.8.0.2/24", "tunnel address CIDR")
		socks := fs.String("socks", "", "local SOCKS5 listen (legacy mode; skips TUN)")
		route := fs.String("route", "", "extra CIDRs to route via tunnel gateway (comma-separated)")
		_ = fs.Parse(os.Args[2:])
		if *server == "" {
			log.Fatal("need -server host:port")
		}
		if *key == "" {
			log.Fatal("need -key or CLOAK_KEY")
		}
		if *socks != "" {
			if err := runSOCKSClient(*server, *key, *socks); err != nil {
				log.Fatal(err)
			}
			return
		}
		k, err := parseKey(*key)
		if err != nil {
			log.Fatal(err)
		}
		if err := runVPNClient(*server, vpnOpts{
			key:     k,
			tunName: *tunName,
			cidr:    *addr,
			routes:  splitCSV(*route),
		}); err != nil {
			log.Fatal(err)
		}
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `cloak — layer-3 VPN over encrypted UDP

  cloak server -listen :51820 -key <psk> -tun cloak0 -addr 10.8.0.1/24
  cloak client -server host:51820 -key <psk> -tun cloak0 -addr 10.8.0.2/24

Needs admin/sudo for the TUN. On Windows, put wintun.dll next to cloak.exe
(from https://www.wintun.net/).

Optional -route 192.168.0.0/16 sends that CIDR through the tunnel gateway
(first host in -addr's subnet, usually 10.8.0.1).

Legacy SOCKS (no TUN): 
  cloak server -listen :51820 -key <psk> -socks
  cloak client -server host:51820 -key <psk> -socks 127.0.0.1:1080

Key: 64-char hex or passphrase (sha256). Or CLOAK_KEY.
`)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
