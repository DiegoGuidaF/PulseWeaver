package httpserver

import (
	"fmt"
	"net/netip"
	"strings"
)

// ParseTrustedProxy parses a single plain IP address (IPv4 or IPv6).
// Returns an error if the IP is invalid, contains CIDR notation, or contains commas.
func ParseTrustedProxy(s string) (netip.Addr, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return netip.Addr{}, nil
	}

	addr, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("TRUSTED_PROXY must be a single plain IP address (no CIDR notation, no comma-separated lists). Got: '%s'", s)
	}

	return addr, nil
}
