package hostaccess

import (
	"errors"
	"fmt"
	"strings"
)

// ErrBadRequest is a sentinel wrapping input-validation failures that handlers should map to HTTP 400.
var ErrBadRequest = errors.New("bad request")

// BulkCreateKnownHostsParams holds validated, normalised FQDNs for bulk creation.
type BulkCreateKnownHostsParams struct {
	FQDNs []string
}

// NewBulkCreateKnownHostsParams trims, lowercases, and deduplicates the input list.
// Returns an error if the result is empty or any element normalises to "".
// TODO: Should there be any validation on the fqdn being valid?
func NewBulkCreateKnownHostsParams(fqdns []string) (BulkCreateKnownHostsParams, error) {
	if len(fqdns) == 0 {
		return BulkCreateKnownHostsParams{}, fmt.Errorf("%w: at least one FQDN required", ErrBadRequest)
	}
	seen := make(map[string]struct{}, len(fqdns))
	out := make([]string, 0, len(fqdns))
	for _, raw := range fqdns {
		f := strings.ToLower(strings.TrimSpace(raw))
		//TODO: Should we be this strict?
		if f == "" {
			return BulkCreateKnownHostsParams{}, fmt.Errorf("%w: blank FQDN in request", ErrBadRequest)
		}
		if _, dup := seen[f]; dup {
			continue // silently deduplicate within a single request
		}
		//TODO: Is this the usual Go way of doing this deduplication?
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return BulkCreateKnownHostsParams{FQDNs: out}, nil
}
