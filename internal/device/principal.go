package device

import "github.com/DiegoGuidaF/PulseWeaver/internal/ids"

type Principal struct {
	DeviceID ids.DeviceID
}

func NewPrincipal(deviceID ids.DeviceID) *Principal {
	return &Principal{
		DeviceID: deviceID,
	}
}

func PrincipalFromDevice(device *Device) *Principal {
	return NewPrincipal(device.ID)
}
