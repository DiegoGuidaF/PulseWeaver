package networkpolicies

import (
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// NetworkPolicy is the core entity.
type NetworkPolicy struct {
	ID              ids.NetworkPolicyID
	Name            string
	CIDR            string // normalized ("192.168.1.0/24"), never raw user input
	Description     *string
	Enabled         bool
	BypassHostCheck bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CacheEntry is consumed by the policy package via its NetworkPoliciesProvider interface.
type CacheEntry struct {
	PolicyID         ids.NetworkPolicyID
	PolicyName       string
	CIDR             string
	AllowAllHosts    bool
	AllowedHostFQDNs []string // empty = deny-all (only meaningful when AllowAllHosts=false)
}

// UpdateFields contains only the fields to update; nil means "unchanged".
// Description uses **string so callers can express "set to null" vs "not provided".
type UpdateFields struct {
	Name        string
	CIDR        string
	Description string
	Enabled     bool
}

// Apply merges fields onto p, normalizing CIDR, and returns the result.
func (p NetworkPolicy) Apply(fields UpdateFields) (NetworkPolicy, error) {
	updated := p
	updated.Name = fields.Name
	normalized, err := normalizeCIDR(fields.CIDR)
	if err != nil {
		return NetworkPolicy{}, fmt.Errorf("%w: %s", ErrInvalidCIDR, fields.CIDR)
	}
	updated.CIDR = normalized
	updated.Description = new(fields.Description)
	updated.Enabled = fields.Enabled
	return updated, nil
}

// normalizeCIDR returns the network address in CIDR notation (host bits zeroed).
func normalizeCIDR(cidr string) (string, error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return "", err
	}
	return prefix.Masked().String(), nil
}

// Sentinel errors.
var (
	ErrNotFound     = errors.New("network policy not found")
	ErrCIDRConflict = errors.New("a policy with this CIDR already exists")
	ErrInvalidCIDR  = errors.New("invalid CIDR notation")
)
