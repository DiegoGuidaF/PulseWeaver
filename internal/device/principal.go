package device

type Principal struct {
	DeviceID DeviceID
}

func NewPrincipal(deviceID DeviceID) *Principal {
	return &Principal{
		DeviceID: deviceID,
	}
}

func PrincipalFromDevice(device *Device) *Principal {
	return NewPrincipal(device.ID)
}
