package registration

import (
	"fmt"
	"strconv"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// PendingRegistrationID is the primary key of a pending registration.
type PendingRegistrationID int64

func (id PendingRegistrationID) Int64() int64 {
	return int64(id)
}

func (id PendingRegistrationID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// PendingRegistration holds an invite created by an admin before a device is claimed.
type PendingRegistration struct {
	ID PendingRegistrationID `db:"id"`

	DeviceName string     `db:"device_name"`
	OwnerID    ids.UserID `db:"owner_id"`

	// RegistrationCode is the full base64url-encoded code delivered to the app.
	// Nulled after claim.
	RegistrationCode *string `db:"registration_code"`

	HeartbeatServerURL       string `db:"heartbeat_server_url"`
	HeartbeatIntervalSeconds int    `db:"heartbeat_interval_seconds"`
	AppBiometricEnabled      bool   `db:"app_biometric_enabled"`
	AppSettingsLocked        bool   `db:"app_settings_locked"`

	ExpiresAt time.Time  `db:"expires_at"`
	CreatedAt time.Time  `db:"created_at"`
	UsedAt    *time.Time `db:"used_at"`

	// InvalidatedAt is set when an admin invalidates a pending invite (soft delete).
	InvalidatedAt *time.Time `db:"invalidated_at"`

	// CreatedDeviceID is set after the invite is successfully claimed.
	CreatedDeviceID *int64 `db:"created_device_id"`
}

// Status derives the lifecycle status from the record fields.
func (p *PendingRegistration) Status() PendingRegistrationStatus {
	if p.UsedAt != nil {
		return StatusUsed
	}
	if p.InvalidatedAt != nil {
		return StatusInvalidated
	}
	if time.Now().After(p.ExpiresAt) {
		return StatusExpired
	}
	return StatusPending
}

func (p *PendingRegistration) ToClaimResult(rawAPIKey string) ClaimResult {
	return ClaimResult{
		p.HeartbeatServerURL,
		p.HeartbeatIntervalSeconds,
		p.AppBiometricEnabled,
		p.AppSettingsLocked,
		rawAPIKey,
	}

}

// PendingRegistrationStatus is the derived lifecycle state of an invite.
type PendingRegistrationStatus string

const (
	StatusPending     PendingRegistrationStatus = "pending"
	StatusUsed        PendingRegistrationStatus = "used"
	StatusExpired     PendingRegistrationStatus = "expired"
	StatusInvalidated PendingRegistrationStatus = "invalidated"
)

// CreateInviteRequest carries the admin-provided fields for a new invite.
type CreateInviteRequest struct {
	DeviceName          string
	OwnerID             ids.UserID
	HeartbeatServerURL  string
	IntervalSeconds     int
	AppBiometricEnabled bool
	AppSettingsLocked   bool
	ExpiresInHours      int
	ExpiresAt           time.Time
	RegistrationCode    string
}

func (c *CreateInviteRequest) addRegistrationCode() error {
	code, _, err := generateRegistrationCode(c.HeartbeatServerURL)
	if err != nil {
		return fmt.Errorf("generate registration code: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(c.ExpiresInHours) * time.Hour)

	c.RegistrationCode = code
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

// InviteFilter controls which invites are returned by ListInvites.
type InviteFilter struct {
	// IncludeAll includes used and expired invites in addition to pending ones.
	IncludeAll bool
}
