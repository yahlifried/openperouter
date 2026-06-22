// SPDX-License-Identifier:Apache-2.0

package staticconfiguration

import (
	"net"
	"testing"

	"github.com/openperouter/openperouter/api/static"
)

func TestNodeIndexValidate(t *testing.T) {
	tests := []struct {
		name        string
		nodeIndex   static.NodeIndex
		expectError bool
	}{
		{
			name:      "index only",
			nodeIndex: static.NodeIndex{Index: 42},
		},
		{
			name:      "interface only",
			nodeIndex: static.NodeIndex{InterfaceName: "eth0"},
		},
		{
			name:      "neither set",
			nodeIndex: static.NodeIndex{},
		},
		{
			name:        "both set is an error",
			nodeIndex:   static.NodeIndex{Index: 5, InterfaceName: "eth0"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNodeIndex(tt.nodeIndex)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNodeIndexFromInterface(t *testing.T) {
	loopback := loopbackInterfaceName(t)

	t.Run("loopback resolves to 1", func(t *testing.T) {
		result, err := NodeIndexFromInterface(loopback)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 1 {
			t.Errorf("NodeIndexFromInterface(%s) = %d, want 1", loopback, result)
		}
	})

	t.Run("non-existent interface", func(t *testing.T) {
		_, err := NodeIndexFromInterface("nonexistent-iface-xyz")
		if err == nil {
			t.Error("expected error but got none")
		}
	})
}

func TestHostPartFromIPNet(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		expected int
	}{
		{
			name:     "/24 host part 80",
			cidr:     "192.168.111.80/24",
			expected: 80,
		},
		{
			name:     "/24 host part 1",
			cidr:     "10.0.0.1/24",
			expected: 1,
		},
		{
			name:     "/24 host part 254",
			cidr:     "172.16.0.254/24",
			expected: 254,
		},
		{
			name:     "/16 host part",
			cidr:     "10.5.3.7/16",
			expected: 3*256 + 7,
		},
		{
			name:     "/28 host part",
			cidr:     "192.168.1.67/28",
			expected: 3,
		},
		{
			name:     "/32 host part is always 0",
			cidr:     "10.0.0.5/32",
			expected: 0,
		},
		{
			name:     "/8 host part",
			cidr:     "10.1.2.3/8",
			expected: 1*65536 + 2*256 + 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, ipNet, err := net.ParseCIDR(tt.cidr)
			if err != nil {
				t.Fatalf("failed to parse CIDR %s: %v", tt.cidr, err)
			}
			ipNet.IP = ip

			result := hostPartFromIPNet(ipNet)
			if result != tt.expected {
				t.Errorf("hostPartFromIPNet(%s) = %d, want %d", tt.cidr, result, tt.expected)
			}
		})
	}

	t.Run("16-byte IPv4 mask", func(t *testing.T) {
		ipNet := &net.IPNet{
			IP:   net.IPv4(192, 168, 1, 42),
			Mask: net.CIDRMask(24, 128)[:16],
		}
		// CIDRMask(24, 128) gives a 16-byte mask; the last 4 bytes are
		// all zeros, so after slicing to [12:] we get 255.255.255.0.
		// However, CIDRMask(24, 128) sets the first 24 bits of 128,
		// which means bytes 0-2 are 0xFF and bytes 3-15 are 0x00.
		// After our fix slices mask[12:] we get [0,0,0,0] → host part is full IP.
		// Instead, simulate a realistic 16-byte mask that iface.Addrs() would return:
		// an IPv4-mapped-in-IPv6 mask where the IPv4 /24 portion sits in the last 4 bytes.
		ipNet.Mask = make(net.IPMask, 16)
		copy(ipNet.Mask[12:], net.CIDRMask(24, 32))

		result := hostPartFromIPNet(ipNet)
		if result != 42 {
			t.Errorf("hostPartFromIPNet with 16-byte mask = %d, want 42", result)
		}
	})
}

func loopbackInterfaceName(t *testing.T) string {
	t.Helper()
	for _, name := range []string{"lo", "lo0"} {
		if _, err := net.InterfaceByName(name); err == nil {
			return name
		}
	}
	t.Skip("no loopback interface found")
	return ""
}
