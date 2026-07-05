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
	DenyReasonNoDeviceMatch   DenyReason = DenyReason(httpapi.PolicyDenyReasonNoDeviceMatch)
	DenyReasonIPNotRegistered DenyReason = DenyReason(httpapi.PolicyDenyReasonIpNotRegistered)
	DenyReasonInvalidToken    DenyReason = DenyReason(httpapi.PolicyDenyReasonInvalidToken)
	DenyReasonHostNotAllowed  DenyReason = DenyReason(httpapi.PolicyDenyReasonHostNotAllowed)
)

// MatchSource identifies which mechanism authorized a verify request.
type MatchSource string

const (
	MatchSourceDevice        MatchSource = MatchSource(httpapi.PolicyMatchSourceDevice)
	MatchSourceNetworkPolicy MatchSource = MatchSource(httpapi.PolicyMatchSourceNetworkPolicy)
)
