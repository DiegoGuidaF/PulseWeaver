package policy

import (
	"context"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
)

// DecisionObserver is implemented by any component that wants to react to
// every access-control decision made by the policy service.
type DecisionObserver interface {
	OnDecision(ctx context.Context, event DecisionEvent)
}

// IPContributor records one device/address/user triple that contributed to the
// IP set entry used for an access decision.
type IPContributor struct {
	DeviceID  device.DeviceID
	AddressID device.AddressID
	UserID    auth.UserID
}

type DecisionEvent struct {
	ClientIP       string
	Outcome        bool
	DenyReason     *DenyReason
	IPContributors []IPContributor // nil if IP not found; ≥1 on allow or host-denied
	CreatedAt      time.Time
	DurationUs     int64
	TargetHost     *string
	TargetURI      *string
	HTTPMethod     *string
	XFFChain       *string
	Headers        map[string][]string
	GeoIP          geoip.Result
}

type DenyReason string

const (
	DenyReasonNoDeviceMatch   DenyReason = "no_device_match"
	DenyReasonIPNotRegistered DenyReason = "ip_not_registered"
	DenyReasonInvalidToken    DenyReason = "invalid_token"
	DenyReasonHostNotAllowed  DenyReason = "host_not_allowed"
)

func NewDecisionEvent(outcome bool, denyReason *DenyReason, contributors []IPContributor, req *VerifyRequest, geo geoip.Result, durationUs int64) DecisionEvent {
	headers := req.Headers
	if req.Headers == nil {
		headers = make(map[string][]string)
	}
	return DecisionEvent{
		ClientIP:       req.ClientIP,
		Outcome:        outcome,
		DenyReason:     denyReason,
		IPContributors: contributors,
		CreatedAt:      time.Now().UTC(),
		DurationUs:     durationUs,
		TargetHost:     req.TargetHost,
		TargetURI:      req.TargetURI,
		HTTPMethod:     req.HTTPMethod,
		XFFChain:       req.XFFChain,
		Headers:        headers,
		GeoIP:          geo,
	}
}
