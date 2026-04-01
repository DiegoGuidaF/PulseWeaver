package rule

import "errors"

var (
	// ErrRuleNotFound is returned when a requested rule does not exist.
	ErrRuleNotFound = errors.New("rule not found")

	// ErrInvalidRuleConfig is returned when a rule's configuration cannot be parsed
	// or violates basic invariants.
	ErrInvalidRuleConfig = errors.New("invalid rule config")

	// ErrInvalidTTL is returned when a AddressTTLSeconds value is non-positive.
	ErrInvalidTTL = errors.New("ttl_seconds must be greater than 0")

	// ErrInvalidMaxAddresses is returned when max_addresses is less than 1.
	ErrInvalidMaxAddresses = errors.New("max_addresses must be at least 1")
)
