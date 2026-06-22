// SPDX-License-Identifier:Apache-2.0

package staticconfiguration

import (
	"fmt"
	"net"

	"github.com/openperouter/openperouter/api/static"
)

func ValidateNodeIndex(n static.NodeIndex) error {
	if n.Index != 0 && n.InterfaceName != "" {
		return fmt.Errorf(
			"index and interfaceName are mutually exclusive, got index %d and interface %q",
			n.Index, n.InterfaceName)
	}
	return nil
}

func NodeIndexFromInterface(name string) (int, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return 0, fmt.Errorf("failed to find interface %s: %w", name, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return 0, fmt.Errorf("failed to get addresses for interface %s: %w", name, err)
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ipNet.IP.To4() == nil {
			continue
		}
		return hostPartFromIPNet(ipNet), nil
	}

	return 0, fmt.Errorf("no IPv4 address found on interface %s", name)
}

func hostPartFromIPNet(ipNet *net.IPNet) int {
	ip4 := ipNet.IP.To4()
	if ip4 == nil {
		return 0
	}

	mask := ipNet.Mask
	if len(mask) == 16 {
		mask = mask[12:]
	}
	if len(mask) < 4 {
		return 0
	}

	hostPart := 0
	for i := range 4 {
		hostPart = (hostPart << 8) | int(ip4[i] & ^mask[i])
	}

	return hostPart
}
