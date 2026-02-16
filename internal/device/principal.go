package device

type Principal struct {
	DeviceID DeviceID
}

func NewPrincipal(deviceId DeviceID) *Principal {
	return &Principal{
		DeviceID: deviceId,
	}
}

func PrincipalFromDevice(device *Device) *Principal {
	return NewPrincipal(device.ID)
}
