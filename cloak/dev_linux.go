//go:build linux

package main

import (
	"fmt"
	"os/exec"

	"golang.zx2c4.com/wireguard/tun"
)

func configureDevice(dev tun.Device, cidr string) error {
	name, err := dev.Name()
	if err != nil {
		return err
	}
	if out, err := exec.Command("ip", "link", "set", "dev", name, "up").CombinedOutput(); err != nil {
		return fmt.Errorf("ip link up: %w: %s", err, out)
	}
	if out, err := exec.Command("ip", "addr", "add", cidr, "dev", name).CombinedOutput(); err != nil {
		return fmt.Errorf("ip addr add: %w: %s", err, out)
	}
	return nil
}

func addRouteVia(dev tun.Device, destCIDR, gateway string) error {
	name, err := dev.Name()
	if err != nil {
		return err
	}
	out, err := exec.Command("ip", "route", "add", destCIDR, "via", gateway, "dev", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ip route add: %w: %s", err, out)
	}
	return nil
}
