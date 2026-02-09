package device

import "errors"

var (
	ErrDeviceNotFound          = errors.New("device not found")
	ErrAddressNotFound         = errors.New("device address not found")
	ErrInvalidIPFormat         = errors.New("invalid IP address format")
	ErrAddressNotOwnedByDevice = errors.New("address is not owned by the device")
)
