//go:build darwin

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
	// utun on mac: ifconfig NAME inet LOCAL PEER netmask MASK up
	peer := subnetPeer(v4, ipnet)
	mask := net.IP(ipnet.Mask).String()
	out, err := exec.Command("ifconfig", name, "inet", v4.String(), peer.String(), "netmask", mask, "up").CombinedOutput()
	if err != nil {
		return fmt.Errorf("ifconfig: %w: %s", err, out)
	}
	return nil
}

func addRouteVia(dev tun.Device, destCIDR, gateway string) error {
	name, err := dev.Name()
	if err != nil {
		return err
	}
	out, err := exec.Command("route", "-n", "add", "-net", destCIDR, gateway, "-iface", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("route add: %w: %s", err, out)
	}
	return nil
}

func subnetPeer(local net.IP, n *net.IPNet) net.IP {
	gw, err := subnetGateway(n.String())
	if err != nil {
		return local
	}
	if gw.Equal(local) {
		// pick another host if we are .1
		p := make(net.IP, 4)
		copy(p, local.To4())
		p[3]++
		return p
	}
	return gw
}
