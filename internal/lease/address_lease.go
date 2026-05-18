package lease

import (
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

type AddressLease struct {
	ID        ids.AddressLeaseID `db:"id"`
	DeviceID  ids.DeviceID       `db:"device_id"`
	AddressID ids.AddressID      `db:"address_id"`
	ExpiresAt *time.Time         `db:"expires_at"`
	UpdatedAt time.Time          `db:"updated_at"`
	CreatedAt time.Time          `db:"created_at"`
}

// NewAddressLease builds an AddressLease.
// expiresAt is nil when no addressTTL is nil.
func NewAddressLease(addressID ids.AddressID, deviceID ids.DeviceID, addressTTL *int) AddressLease {
	now := time.Now().UTC()
	return AddressLease{
		AddressID: addressID,
		DeviceID:  deviceID,
		ExpiresAt: expiresAtFromTTL(now, addressTTL),
		UpdatedAt: now,
		CreatedAt: now,
	}
}

// expiresAtFromTTL returns now+TTL or nil if TTL is nil
func expiresAtFromTTL(now time.Time, addressTTL *int) *time.Time {
	if addressTTL != nil {
		duration := time.Duration(*addressTTL) * time.Second
		return new(now.Add(duration))
	}

	return nil

}
