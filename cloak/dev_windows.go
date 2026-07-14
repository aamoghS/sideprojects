//go:build windows

package main

import (
	"fmt"
	"net"
	"os/exec"

	"golang.zx2c4.com/wireguard/tun"
)

func configureDevice(dev tun.Device, cidr string) error {
	name, err := dev.Name()
	if err != nil {
		return err
	}
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	v4 := ip.To4()
	if v4 == nil {
		return fmt.Errorf("cloak: IPv4 -addr required")
	}
	mask := net.IP(ipnet.Mask).String()
	cmd := exec.Command("netsh", "interface", "ip", "set", "address",
		"name="+name, "source=static", "addr="+v4.String(), "mask="+mask)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh set address: %w: %s", err, bytesTrim(out))
	}
	return nil
}

func addRouteVia(dev tun.Device, destCIDR, gateway string) error {
	name, err := dev.Name()
	if err != nil {
		return err
	}
	cmd := exec.Command("netsh", "interface", "ipv4", "add", "route",
		destCIDR, name, gateway, "metric=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh add route: %w: %s", err, bytesTrim(out))
	}
	return nil
}

func bytesTrim(b []byte) string {
	s := string(b)
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
