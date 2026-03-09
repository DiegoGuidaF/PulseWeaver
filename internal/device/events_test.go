//go:build test

package device

import (
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestAddressEvent_IsAddressEnabled(t *testing.T) {
	is := is.New(t)
	base := AddressEvent{
		AddressID:  AddressID(1),
		DeviceID:   DeviceID(1),
		OccurredAt: time.Now().UTC(),
	}

	is.True(AddressEvent{Type: EventTypeAddressCreated, AddressID: base.AddressID, DeviceID: base.DeviceID, OccurredAt: base.OccurredAt}.IsAddressEnabled())
	is.True(AddressEvent{Type: EventTypeAddressEnabled, AddressID: base.AddressID, DeviceID: base.DeviceID, OccurredAt: base.OccurredAt}.IsAddressEnabled())
	is.True(AddressEvent{Type: EventTypeAddressRefreshed, AddressID: base.AddressID, DeviceID: base.DeviceID, OccurredAt: base.OccurredAt}.IsAddressEnabled())
	is.True(!AddressEvent{Type: EventTypeAddressDisabled, AddressID: base.AddressID, DeviceID: base.DeviceID, OccurredAt: base.OccurredAt}.IsAddressEnabled())
	is.True(!AddressEvent{Type: EventType(""), AddressID: base.AddressID, DeviceID: base.DeviceID, OccurredAt: base.OccurredAt}.IsAddressEnabled())
}
