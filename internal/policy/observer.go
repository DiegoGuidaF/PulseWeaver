package policy

import (
	"context"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
)

// DecisionObserver is implemented by any component that wants to react to
// every access-control decision made by the policy service.
type DecisionObserver interface {
	OnDecision(ctx context.Context, event DecisionEvent)
}

type DecisionEvent struct {
	ClientIP   string
	Outcome    bool
	DenyReason *DenyReason
	DeviceID   *device.DeviceID
	AddressID  *device.AddressID
	CreatedAt  time.Time
	DurationUs int64
	TargetHost *string
	TargetURI  *string
	HTTPMethod *string
	XFFChain   *string
	Headers    map[string][]string
	GeoIP      geoip.Result
}

type DenyReason string

const (
	DenyReasonNoDeviceMatch   DenyReason = "no_device_match"
	DenyReasonIPNotRegistered DenyReason = "ip_not_registered"
	DenyReasonInvalidToken    DenyReason = "invalid_token"
)

func NewDecisionEvent(outcome bool, denyReason *DenyReason, deviceID *device.DeviceID, addressID *device.AddressID, req *VerifyRequest, geo geoip.Result, durationUs int64) DecisionEvent {
	// Ensure headers map is non-nil.
	headers := req.Headers
	if req.Headers == nil {
		headers = make(map[string][]string)
	}
	return DecisionEvent{
		ClientIP:   req.ClientIP,
		Outcome:    outcome,
		DenyReason: denyReason,
		DeviceID:   deviceID,
		AddressID:  addressID,
		CreatedAt:  time.Now().UTC(),
		DurationUs: durationUs,
		TargetHost: req.TargetHost,
		TargetURI:  req.TargetURI,
		HTTPMethod: req.HTTPMethod,
		XFFChain:   req.XFFChain,
		Headers:    headers,
		GeoIP:      geo,
	}
}
