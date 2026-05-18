package networkpolicies

import "context"

// PolicyChangeObserver is implemented by consumers that must react when any
// network policy is created, updated, deleted, or has its host access changed.
// Producer declares; consumers implement.
type PolicyChangeObserver interface {
	OnNetworkPolicyChanged(ctx context.Context)
}
