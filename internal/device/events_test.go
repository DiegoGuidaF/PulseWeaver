//go:build test

package device_test

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/matryer/is"
)

func TestAddressEvent_IsAddressEnabled(t *testing.T) {
	is := is.New(t)
	base := device.AddressEvent{
		AddressID:  device.AddressID(1),
		DeviceID:   device.DeviceID(1),
		OccurredAt: time.Now().UTC(),
	}

	is.True(device.AddressEvent{Type: device.EventTypeAddressCreated, AddressID: base.AddressID, DeviceID: base.DeviceID, OccurredAt: base.OccurredAt}.IsAddressEnabled())
	is.True(device.AddressEvent{Type: device.EventTypeAddressEnabled, AddressID: base.AddressID, DeviceID: base.DeviceID, OccurredAt: base.OccurredAt}.IsAddressEnabled())
	is.True(device.AddressEvent{Type: device.EventTypeAddressRefreshed, AddressID: base.AddressID, DeviceID: base.DeviceID, OccurredAt: base.OccurredAt}.IsAddressEnabled())
	is.True(!device.AddressEvent{Type: device.EventTypeAddressDisabled, AddressID: base.AddressID, DeviceID: base.DeviceID, OccurredAt: base.OccurredAt}.IsAddressEnabled())
	is.True(!device.AddressEvent{Type: "", AddressID: base.AddressID, DeviceID: base.DeviceID, OccurredAt: base.OccurredAt}.IsAddressEnabled())
}
