package hostaccess

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
)

// ErrBadRequest is a sentinel wrapping input-validation failures that handlers should map to HTTP 400.
var ErrBadRequest = errors.New("bad request")

// BulkCreateKnownHostsParams holds validated, normalised FQDNs for bulk creation.
type BulkCreateKnownHostsParams struct {
	FQDNs []string
}

// NewBulkCreateKnownHostsParams trims, lowercases, and deduplicates the input list.
// Returns an error if the result is empty or any element normalises to "".
func NewBulkCreateKnownHostsParams(fqdns []string) (BulkCreateKnownHostsParams, error) {
	if len(fqdns) == 0 {
		return BulkCreateKnownHostsParams{}, fmt.Errorf("%w: at least one FQDN required", ErrBadRequest)
	}
	seen := make(map[string]struct{}, len(fqdns))
	out := make([]string, 0, len(fqdns))
	for _, raw := range fqdns {
		f := strings.ToLower(strings.TrimSpace(raw))
		if f == "" {
			return BulkCreateKnownHostsParams{}, fmt.Errorf("%w: blank FQDN in request", ErrBadRequest)
		}
		if _, dup := seen[f]; dup {
			continue // silently deduplicate within a single request
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return BulkCreateKnownHostsParams{FQDNs: out}, nil
}

// SetHostGroupMembersParams holds a deduplicated list of host IDs for a group.
type SetHostGroupMembersParams struct {
	GroupID HostGroupID
	HostIDs []KnownHostID
}

// NewSetHostGroupMembersParams deduplicates the host ID list.
func NewSetHostGroupMembersParams(groupID HostGroupID, hostIDs []KnownHostID) SetHostGroupMembersParams {
	seen := make(map[KnownHostID]struct{}, len(hostIDs))
	out := make([]KnownHostID, 0, len(hostIDs))
	for _, id := range hostIDs {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return SetHostGroupMembersParams{GroupID: groupID, HostIDs: out}
}

// SetUserGrantsParams holds deduplicated host and group ID lists for a user.
type SetUserGrantsParams struct {
	UserID   auth.UserID
	HostIDs  []KnownHostID
	GroupIDs []HostGroupID
}

// NewSetUserGrantsParams deduplicates both ID lists.
func NewSetUserGrantsParams(userID auth.UserID, hostIDs []KnownHostID, groupIDs []HostGroupID) SetUserGrantsParams {
	seenHosts := make(map[KnownHostID]struct{}, len(hostIDs))
	outHosts := make([]KnownHostID, 0, len(hostIDs))
	for _, id := range hostIDs {
		if _, dup := seenHosts[id]; dup {
			continue
		}
		seenHosts[id] = struct{}{}
		outHosts = append(outHosts, id)
	}

	seenGroups := make(map[HostGroupID]struct{}, len(groupIDs))
	outGroups := make([]HostGroupID, 0, len(groupIDs))
	for _, id := range groupIDs {
		if _, dup := seenGroups[id]; dup {
			continue
		}
		seenGroups[id] = struct{}{}
		outGroups = append(outGroups, id)
	}

	return SetUserGrantsParams{UserID: userID, HostIDs: outHosts, GroupIDs: outGroups}
}
