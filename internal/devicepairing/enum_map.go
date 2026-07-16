package devicepairing

import "github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"

// PairingStatusToAPI converts the domain pairing status to its API
// representation. The exhaustive linter fails this switch when a new
// PairingStatus value is added without a corresponding case here.
func PairingStatusToAPI(s PairingStatus) httpapi.DevicePairingStatus {
	switch s {
	case StatusPending:
		return httpapi.DevicePairingStatusPending
	case StatusUsed:
		return httpapi.DevicePairingStatusUsed
	case StatusExpired:
		return httpapi.DevicePairingStatusExpired
	case StatusInvalidated:
		return httpapi.DevicePairingStatusInvalidated
	case StatusReplaced:
		return httpapi.DevicePairingStatusReplaced
	default:
		return httpapi.DevicePairingStatus(s)
	}
}
