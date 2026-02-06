package device

import "errors"

var (
	ErrDeviceNotFound   = errors.New("device not found")
	ErrAddressNotFound  = errors.New("device address not found")
	ErrInvalidIPFormat  = errors.New("invalid IP address format")
	ErrIPv6NotSupported = errors.New("only IPv4 addresses are supported")
)
