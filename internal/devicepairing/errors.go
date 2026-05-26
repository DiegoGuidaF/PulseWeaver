package devicepairing

import "errors"

var (
	// ErrPairingNotFound is returned when the pairing code or ID does not match any record.
	ErrPairingNotFound = errors.New("device pairing not found")

	// ErrPairingNotPending is returned when the pairing is already used or expired at delete time.
	ErrPairingNotPending = errors.New("device pairing is no longer pending")

	ErrPairingNotClaimable = errors.New("cannot claim an invalid or expired pairing")
	ErrPairingExpired      = errors.New("pairing expired")
)
