package policy

import (
	"context"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// DecisionObserver is implemented by any component that wants to react to
// every access-control decision made by the policy service.
type DecisionObserver interface {
	OnDecision(ctx context.Context, event DecisionEvent)
}

// IPContributor records one device/address/user triple that contributed to the
// IP set entry used for an access decision.
type IPContributor struct {
	DeviceID  ids.DeviceID
	AddressID ids.AddressID
	UserID    ids.UserID
}

// ContributorAccess extends IPContributor with the per-user pre-intersection host access
// state stored on each ipSetEntry for audit and simulate purposes.
type ContributorAccess struct {
	DeviceID         ids.DeviceID
	AddressID        ids.AddressID
	UserID           ids.UserID
	UserBypass       bool
	UserAllowedHosts []string // case-folded; sorted lexicographically
}

type DecisionEvent struct {
	ClientIP          string
	Outcome           bool
	DenyReason        *DenyReason
	IPContributors    []IPContributor // nil if IP not found; ≥1 on allow or host-denied
	MatchSource       MatchSource
	NetworkPolicyID   *ids.NetworkPolicyID
	NetworkPolicyName *string
	CreatedAt         time.Time
	DurationUs        int64
	TargetHost        *string
	TargetURI         *string
	HTTPMethod        *string
	XFFChain          *string
	Headers           map[string][]string
	GeoIP             geoip.Result
}

func NewDecisionEvent(outcome bool, denyReason *DenyReason, result *DecisionResult, req *VerifyRequest, geo geoip.Result, durationUs int64) DecisionEvent {
	headers := req.Headers
	if req.Headers == nil {
		headers = make(map[string][]string)
	}
	e := DecisionEvent{
		ClientIP:   req.ClientIP.String(),
		Outcome:    outcome,
		DenyReason: denyReason,
		CreatedAt:  time.Now().UTC(),
		DurationUs: durationUs,
		TargetHost: req.TargetHost,
		TargetURI:  req.TargetURI,
		HTTPMethod: req.HTTPMethod,
		XFFChain:   req.XFFChain,
		Headers:    headers,
		GeoIP:      geo,
	}
	if result != nil {
		e.IPContributors = result.Contributors
		e.MatchSource = result.MatchSource
		e.NetworkPolicyID = result.NetworkPolicyID
		e.NetworkPolicyName = result.NetworkPolicyName
	}
	return e
}
