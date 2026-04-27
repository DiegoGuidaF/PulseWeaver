package hostaccess

import (
	"errors"
	"fmt"
	"strings"
)

// ErrBadRequest is a sentinel wrapping input-validation failures that handlers should map to HTTP 400.
var ErrBadRequest = errors.New("bad request")

// ValidateFQDN checks that s is a well-formed hostname per RFC 1123.
// It expects the input to already be lowercased/trimmed (NormaliseFQDN does this).
func ValidateFQDN(s string) error {
	// Strip optional trailing dot (absolute DNS form).
	cleaned := strings.TrimSuffix(s, ".")
	if cleaned == "" {
		return fmt.Errorf("%w: FQDN must not be empty", ErrBadRequest)
	}
	if len(cleaned) > 253 {
		return fmt.Errorf("%w: FQDN must not exceed 253 characters", ErrBadRequest)
	}

	labels := strings.Split(cleaned, ".")
	if len(labels) < 2 {
		return fmt.Errorf("%w: FQDN must have at least two labels (e.g. host.example.com)", ErrBadRequest)
	}

	for _, label := range labels {
		if err := validateLabel(label); err != nil {
			return err
		}
	}

	// Reject bare IP addresses: if the last label is all digits, it's not a hostname.
	last := labels[len(labels)-1]
	allDigits := true
	for _, c := range last {
		if c < '0' || c > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		return fmt.Errorf("%w: FQDN must not be a numeric IP address", ErrBadRequest)
	}

	return nil
}

func validateLabel(label string) error {
	if len(label) == 0 || len(label) > 63 {
		return fmt.Errorf("%w: FQDN label must be between 1 and 63 characters", ErrBadRequest)
	}
	if label[0] == '-' || label[len(label)-1] == '-' {
		return fmt.Errorf("%w: FQDN label %q must not start or end with a hyphen", ErrBadRequest, label)
	}
	for _, c := range label {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '-' {
			return fmt.Errorf("%w: FQDN contains invalid character %q", ErrBadRequest, c)
		}
	}
	return nil
}

// NormaliseFQDN trims whitespace, lowercases, and strips a trailing dot.
func NormaliseFQDN(raw string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(raw)), ".")
}
