package devicepairing

import (
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// storedPairingStatus is the subset of PairingStatus values persisted to the DB.
// expired is derived at the business layer and never stored.
type storedPairingStatus string

const (
	storedPending     storedPairingStatus = "pending"
	storedUsed        storedPairingStatus = "used"
	storedInvalidated storedPairingStatus = "invalidated"
	storedReplaced    storedPairingStatus = "replaced"
)

// pairingRow is the raw DB scan target; never leaves the devicepairing package.
type pairingRow struct {
	ID                       ids.DevicePairingID `db:"id"`
	DeviceID                 ids.DeviceID        `db:"device_id"`
	PairingCode              string              `db:"pairing_code"`
	HeartbeatServerURL       string              `db:"heartbeat_server_url"`
	HeartbeatIntervalSeconds int                 `db:"heartbeat_interval_seconds"`
	AppBiometricEnabled      bool                `db:"app_biometric_enabled"`
	AppSettingsLocked        bool                `db:"app_settings_locked"`
	ExpiresAt                time.Time           `db:"expires_at"`
	CreatedAt                time.Time           `db:"created_at"`
	UpdatedAt                time.Time           `db:"updated_at"`
	Status                   storedPairingStatus `db:"status"`
}

// fromRow converts a raw DB row to the domain model, deriving expired status.
func fromRow(r pairingRow) DevicePairing {
	return DevicePairing{
		ID:                       r.ID,
		DeviceID:                 r.DeviceID,
		PairingCode:              r.PairingCode,
		HeartbeatServerURL:       r.HeartbeatServerURL,
		HeartbeatIntervalSeconds: r.HeartbeatIntervalSeconds,
		AppBiometricEnabled:      r.AppBiometricEnabled,
		AppSettingsLocked:        r.AppSettingsLocked,
		ExpiresAt:                r.ExpiresAt,
		CreatedAt:                r.CreatedAt,
		UpdatedAt:                r.UpdatedAt,
		Status:                   EvalStatus(string(r.Status), r.ExpiresAt),
	}
}

// EvalStatus derives the full PairingStatus from a stored status string and expiry time.
// Use this in cross-domain queries that need to surface pairing status without a full
// Repository call (e.g. the device list view).
func EvalStatus(stored string, expiresAt time.Time) PairingStatus {
	if stored == string(storedPending) && time.Now().UTC().After(expiresAt) {
		return StatusExpired
	}
	return PairingStatus(stored)
}

// DevicePairing is the domain model for a pairing record.
type DevicePairing struct {
	ID ids.DevicePairingID

	DeviceID ids.DeviceID

	// PairingCode is the full base64url-encoded code delivered to the app.
	PairingCode string

	HeartbeatServerURL       string
	HeartbeatIntervalSeconds int
	AppBiometricEnabled      bool
	AppSettingsLocked        bool

	ExpiresAt time.Time
	CreatedAt time.Time
	// UpdatedAt is when the status last changed (equals CreatedAt for pending pairings).
	UpdatedAt time.Time

	// Status is the derived lifecycle state; expired is computed from ExpiresAt, never stored.
	Status PairingStatus
}

func (p *DevicePairing) ToClaimResult(rawAPIKey string) ClaimResult {
	return ClaimResult{
		p.HeartbeatServerURL,
		p.HeartbeatIntervalSeconds,
		p.AppBiometricEnabled,
		p.AppSettingsLocked,
		rawAPIKey,
	}
}

// PairingStatus is the full lifecycle state of a device pairing.
type PairingStatus string

const (
	StatusPending     PairingStatus = "pending"
	StatusUsed        PairingStatus = "used"
	StatusExpired     PairingStatus = "expired"
	StatusInvalidated PairingStatus = "invalidated"
	StatusReplaced    PairingStatus = "replaced"
)

// CreatePairingRequest carries the admin-provided fields for a new pairing.
type CreatePairingRequest struct {
	DeviceID            ids.DeviceID
	HeartbeatServerURL  string
	IntervalSeconds     int
	AppBiometricEnabled bool
	AppSettingsLocked   bool
	ExpiresInHours      int
	ExpiresAt           time.Time
	PairingCode         string
}

func (c *CreatePairingRequest) addPairingCode() error {
	code, _, err := generatePairingCode(c.HeartbeatServerURL)
	if err != nil {
		return fmt.Errorf("generate pairing code: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(c.ExpiresInHours) * time.Hour)

	c.PairingCode = code
	c.ExpiresAt = expiresAt
	return nil
}

// ClaimResult is returned after a successful claim: the config payload and the one-time API key.
type ClaimResult struct {
	ServerURL           string
	IntervalSeconds     int
	AppBiometricEnabled bool
	AppSettingsLocked   bool
	RawAPIKey           string // Plaintext — send to app, never stored again.
}

// PairingFilter controls which pairings are returned by ListPairings.
type PairingFilter struct {
	DeviceID ids.DeviceID
	// IncludeAll includes all statuses in addition to pending ones.
	IncludeAll bool
}
