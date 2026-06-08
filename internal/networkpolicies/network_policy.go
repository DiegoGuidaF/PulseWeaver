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
		return NetworkPolicy{}, err
	}
	updated.CIDR = normalized
	updated.Description = fields.Description
	updated.Enabled = fields.Enabled
	return updated, nil
}

// normalizeCIDR returns the network address in CIDR notation (host bits zeroed),
// rejecting prefixes broad enough to cover an entire network operator's range.
func normalizeCIDR(cidr string) (string, error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidCIDR, cidr)
	}
	if classifyCIDR(prefix) == cidrTooBroad {
		return "", fmt.Errorf("%w: %s", ErrCIDRTooBroad, broadLimitMessage(prefix))
	}
	return prefix.Masked().String(), nil
}

// cidrBand classifies how much of the address space a prefix covers.
type cidrBand int

const (
	cidrNormal   cidrBand = iota // narrow enough; no warning
	cidrBroad                    // large but allowed; warrants an audit flag + UI warning
	cidrTooBroad                 // operator-scale (class-A / ISP allocation or broader); rejected
)

// Broadness thresholds, measured on the canonical (unmapped) prefix length.
// IPv4 reasons by host count; IPv6 by allocation structure — every /64 already
// holds 2^64 addresses, so host-count parity is meaningless there. Both reject
// lines mean "an entire network operator's allocation or broader".
const (
	rejectMaxBitsV4 = 8  // /0../8  (>= an entire class-A) is rejected
	warnMaxBitsV4   = 16 // /9../16 warns
	rejectMaxBitsV6 = 32 // /0../32 (an ISP/RIR allocation or broader) is rejected
	warnMaxBitsV6   = 47 // /33../47 warns
)

// classifyCIDR buckets a prefix into normal/broad/too-broad. A 4-in-6 prefix
// (::ffff:a.b.c.d/N) is measured on the IPv4 scale so it shares the IPv4 bands.
func classifyCIDR(prefix netip.Prefix) cidrBand {
	bits := prefix.Bits()
	addr := prefix.Addr()
	if addr.Is4In6() {
		bits -= 96 // drop the 96-bit v6 prefix; measure the embedded v4 prefix
		addr = addr.Unmap()
	}

	rejectMax, warnMax := rejectMaxBitsV6, warnMaxBitsV6
	if addr.Is4() {
		rejectMax, warnMax = rejectMaxBitsV4, warnMaxBitsV4
	}

	switch {
	case bits <= rejectMax:
		return cidrTooBroad
	case bits <= warnMax:
		return cidrBroad
	default:
		return cidrNormal
	}
}

// broadCIDR reports whether a valid (non-rejected) CIDR is large enough to
// warrant an audit flag. Returns false for unparseable input.
func broadCIDR(cidr string) bool {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return false
	}
	return classifyCIDR(prefix) == cidrBroad
}

// broadLimitMessage explains the rejection and names the broadest allowed prefix.
func broadLimitMessage(prefix netip.Prefix) string {
	narrowest := "/9"
	if !prefix.Addr().Unmap().Is4() {
		narrowest = "/33"
	}
	return fmt.Sprintf(
		"%s covers an entire network operator's range; the broadest allowed prefix is %s",
		prefix, narrowest,
	)
}

// Sentinel errors.
var (
	ErrNotFound     = errors.New("network policy not found")
	ErrCIDRConflict = errors.New("a policy with this CIDR already exists")
	ErrInvalidCIDR  = errors.New("invalid CIDR notation")
	ErrCIDRTooBroad = errors.New("CIDR is too broad")
)
