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
	ID              ids.NetworkPolicyID `db:"id"`
	Name            string              `db:"name"`
	CIDR            string              `db:"cidr"` // normalized ("192.168.1.0/24"), never raw user input
	Description     *string             `db:"description"`
	Enabled         bool                `db:"enabled"`
	BypassHostCheck bool                `db:"bypass_host_check"`
	CreatedAt       time.Time           `db:"created_at"`
	UpdatedAt       time.Time           `db:"updated_at"`
}

// CacheEntry is consumed by the policy package via its NetworkPoliciesProvider interface.
type CacheEntry struct {
	PolicyID         ids.NetworkPolicyID
	PolicyName       string
	CIDR             string
	BypassHostCheck  bool
	AllowedHostFQDNs []string // empty = deny-all (only meaningful when BypassHostCheck=false)
}

// UpdateFields contains the fields to update.
// Description uses *string so callers can express "set to null" (nil) vs "keep as empty string".
type UpdateFields struct {
	Name        string
	CIDR        string
	Description *string
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
	updated.Description = fields.Description
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
