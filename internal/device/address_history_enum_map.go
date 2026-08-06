package device

import "github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"

// AddressEventKind classifies an address event against the immediately
// preceding event for the same address, independent of any time window.
type AddressEventKind string

const (
	AddressEventKindCreated  AddressEventKind = "created"
	AddressEventKindEnabled  AddressEventKind = "enabled"
	AddressEventKindDisabled AddressEventKind = "disabled"
	AddressEventKindRefresh  AddressEventKind = "refresh"
)

// AddressEventKindToAPI converts the domain event kind to its API
// representation. The exhaustive linter fails this switch when a new
// AddressEventKind value is added without a corresponding case here.
func AddressEventKindToAPI(k AddressEventKind) httpapi.AddressEventKind {
	switch k {
	case AddressEventKindCreated:
		return httpapi.AddressEventKindCreated
	case AddressEventKindEnabled:
		return httpapi.AddressEventKindEnabled
	case AddressEventKindDisabled:
		return httpapi.AddressEventKindDisabled
	case AddressEventKindRefresh:
		return httpapi.AddressEventKindRefresh
	default:
		return httpapi.AddressEventKind(k)
	}
}

// TTLRisk classifies how an event's renewal gap compares to the device's
// configured lease TTL.
type TTLRisk string

const (
	TTLRiskUnknown     TTLRisk = "unknown"
	TTLRiskOK          TTLRisk = "ok"
	TTLRiskApproaching TTLRisk = "approaching"
	TTLRiskCritical    TTLRisk = "critical"
	TTLRiskBreached    TTLRisk = "breached"
)

// TTLRiskToAPI converts the domain TTL risk to its API representation. The
// exhaustive linter fails this switch when a new TTLRisk value is added
// without a corresponding case here.
func TTLRiskToAPI(r TTLRisk) httpapi.TTLRisk {
	switch r {
	case TTLRiskUnknown:
		return httpapi.Unknown
	case TTLRiskOK:
		return httpapi.Ok
	case TTLRiskApproaching:
		return httpapi.Approaching
	case TTLRiskCritical:
		return httpapi.Critical
	case TTLRiskBreached:
		return httpapi.Breached
	default:
		return httpapi.TTLRisk(r)
	}
}
