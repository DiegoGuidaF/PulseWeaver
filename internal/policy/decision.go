package policy

import (
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// DecisionResult is returned by Decide. It carries enough information for
// VerifyAccess to notify observers and for the simulate handler to build its response.
type DecisionResult struct {
	Allowed           bool
	DenyReason        *DenyReason     // nil when Allowed
	Contributors      []IPContributor // nil when IP not in cache; populated for observer notification
	MatchSource       MatchSource
	NetworkPolicyID   *ids.NetworkPolicyID
	NetworkPolicyName *string
}

// DenyReason identifies why an access request was denied.
type DenyReason string

const (
	DenyReasonNoDeviceMatch   DenyReason = DenyReason(httpapi.NoDeviceMatch)
	DenyReasonIPNotRegistered DenyReason = DenyReason(httpapi.IpNotRegistered)
	DenyReasonInvalidToken    DenyReason = DenyReason(httpapi.InvalidToken)
	DenyReasonHostNotAllowed  DenyReason = DenyReason(httpapi.HostNotAllowed)
)

// MatchSource identifies which mechanism authorized a verify request.
type MatchSource string

const (
	MatchSourceDevice        MatchSource = "device"
	MatchSourceNetworkPolicy MatchSource = "network_policy"
)
