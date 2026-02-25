package rule

import "errors"

var (
	// ErrRuleNotFound is returned when a requested rule does not exist.
	ErrRuleNotFound = errors.New("rule not found")

	// ErrInvalidRuleConfig is returned when a rule's configuration cannot be parsed
	// or violates basic invariants.
	ErrInvalidRuleConfig = errors.New("invalid rule config")
)
