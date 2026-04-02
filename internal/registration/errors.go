package registration

import "errors"

var (
	// ErrInviteNotFound is returned when the registration code or ID does not match any record.
	ErrInviteNotFound = errors.New("registration invite not found")

	// ErrInviteNotPending is returned when the invite is already used or expired at delete time.
	ErrInviteNotPending = errors.New("registration invite is no longer pending")
)

// APIKeyPrefixForTest exposes the device API key prefix for use in tests.
// It mirrors device.APIKeyPrefix so tests can validate the returned key format.
const APIKeyPrefixForTest = "wdk_"
