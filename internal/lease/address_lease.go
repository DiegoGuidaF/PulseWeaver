package lease

import (
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
)

type AddressLease struct {
	ID        AddressLeaseID   `db:"id"`
	AddressID device.AddressID `db:"address_id"`
	ExpiresAt time.Time        `db:"expires_at"`
	UpdatedAt time.Time        `db:"updated_at"`
	CreatedAt time.Time        `db:"created_at"`
}

func NewAddressLease(addressID device.AddressID, duration time.Duration) *AddressLease {
	now := time.Now().UTC()
	expiresAt := now.Add(duration)
	return &AddressLease{
		AddressID: addressID,
		ExpiresAt: expiresAt,
		UpdatedAt: now,
		CreatedAt: now,
	}
}

// AddressLeaseID represents the primary key of a row in the address_leases table.
type AddressLeaseID int64
