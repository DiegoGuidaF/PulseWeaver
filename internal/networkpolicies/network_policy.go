package networkpolicies

import (
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"time"
)

// NetworkPolicyID is a typed alias over int64 for compile-time safety.
type NetworkPolicyID int64

func (id NetworkPolicyID) Int64() int64   { return int64(id) }
func (id NetworkPolicyID) String() string { return strconv.FormatInt(int64(id), 10) }

// NetworkPolicy is the core entity.
type NetworkPolicy struct {
	ID            NetworkPolicyID
	Name          string
	CIDR          string // normalized ("192.168.1.0/24"), never raw user input
	Description   *string
	Enabled       bool
	AllowAllHosts bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CacheEntry is consumed by the policy package via its NetworkPoliciesProvider interface.
type CacheEntry struct {
	PolicyID         NetworkPolicyID
	PolicyName       string
	CIDR             string
	AllowAllHosts    bool
	AllowedHostFQDNs []string // empty = deny-all (only meaningful when AllowAllHosts=false)
}

// UpdateFields contains only the fields to update; nil means "unchanged".
// Description uses **string so callers can express "set to null" vs "not provided".
type UpdateFields struct {
	Name        *string
	CIDR        *string
	Description **string
	Enabled     *bool
}

// Apply merges fields onto p, normalizing CIDR if provided, and returns the result.
// It is the model's responsibility to validate and parse incoming data.
func (p NetworkPolicy) Apply(fields UpdateFields) (NetworkPolicy, error) {
	updated := p
	if fields.Name != nil {
		updated.Name = *fields.Name
	}
	if fields.CIDR != nil {
		normalized, err := normalizeCIDR(*fields.CIDR)
		if err != nil {
			return NetworkPolicy{}, fmt.Errorf("%w: %s", ErrInvalidCIDR, *fields.CIDR)
		}
		updated.CIDR = normalized
	}
	if fields.Description != nil {
		updated.Description = *fields.Description
	}
	if fields.Enabled != nil {
		updated.Enabled = *fields.Enabled
	}
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
	ErrBadRequest   = errors.New("bad request")
)
