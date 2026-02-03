package device

import "errors"

var (
	ErrDeviceNotFound      = errors.New("device not found")
	ErrDeviceIPNotFound    = errors.New("device IP not found")
	ErrDeviceIPDisabled    = errors.New("device IP already disabled")
	ErrDeviceIPWrongDevice = errors.New("device IP does not belong to device")
	ErrInvalidIPFormat     = errors.New("invalid IP address format")
	ErrIPv6NotSupported    = errors.New("only IPv4 addresses are supported")
)
