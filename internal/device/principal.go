package device

import (
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/auth"
)

type Principal struct {
	DeviceID DeviceID
	Role     auth.Role
}

func NewPrincipal(deviceId DeviceID, role auth.Role) *Principal {
	return &Principal{
		DeviceID: deviceId,
		Role:     role,
	}
}

func PrincipalFromDevice(device *Device) *Principal {
	return NewPrincipal(device.ID, auth.DeviceRole)
}
