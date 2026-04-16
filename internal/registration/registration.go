package registration

import (
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
)

// RegistrationID is the primary key of a pending registration.
type RegistrationID = string

// PendingRegistration holds an invite created by an admin before a device is claimed.
type PendingRegistration struct {
	ID string

	DeviceName string
	OwnerID    auth.UserID

	// RegistrationCode is the full base64url-encoded code delivered to the app.
	// Nulled after claim.
	RegistrationCode *string

	// DeviceAPIKey is the pre-generated raw device key stored in plaintext until claimed.
	// Nulled after claim.
	DeviceAPIKey *string

	// DeviceAPIKeyPrefix is kept after claim for admin reference.
	DeviceAPIKeyPrefix string

	HeartbeatServerURL  string
	IntervalSeconds     int
	AppBiometricEnabled bool
	AppSettingsLocked   bool

	ExpiresAt time.Time
	CreatedAt time.Time
	UsedAt    *time.Time

	// CreatedDeviceID is set after the invite is successfully claimed.
	CreatedDeviceID *int64
}

// Status derives the lifecycle status from the record fields.
func (p *PendingRegistration) Status() PendingRegistrationStatus {
	if p.UsedAt != nil {
		return StatusUsed
	}
	if time.Now().After(p.ExpiresAt) {
		return StatusExpired
	}
	return StatusPending
}

// PendingRegistrationStatus is the derived lifecycle state of an invite.
type PendingRegistrationStatus string

const (
	StatusPending PendingRegistrationStatus = "pending"
	StatusUsed    PendingRegistrationStatus = "used"
	StatusExpired PendingRegistrationStatus = "expired"
)

// CreateInviteRequest carries the admin-provided fields for a new invite.
type CreateInviteRequest struct {
	DeviceName          string
	OwnerID             auth.UserID
	HeartbeatServerURL  string
	IntervalSeconds     int
	AppBiometricEnabled bool
	AppSettingsLocked   bool
	ExpiresInHours      int
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
