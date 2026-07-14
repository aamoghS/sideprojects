//go:build !windows && !linux && !darwin

package main

import (
	"fmt"

	"golang.zx2c4.com/wireguard/tun"
)

func configureDevice(dev tun.Device, cidr string) error {
	return fmt.Errorf("cloak: TUN address config not supported on this OS")
}

func addRouteVia(dev tun.Device, destCIDR, gateway string) error {
	return fmt.Errorf("cloak: routes not supported on this OS")
}
