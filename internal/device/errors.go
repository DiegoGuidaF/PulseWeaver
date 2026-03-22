package device

import "errors"

var (
	ErrDeviceNotFound          = errors.New("device not found")
	ErrDuplicateDeviceName     = errors.New("device name already in use")
	ErrAddressNotFound         = errors.New("device address not found")
	ErrInvalidIPFormat         = errors.New("invalid IP address format")
	ErrInvalidDeviceIP         = errors.New("ip address is not valid for device registration")
	ErrTrustedProxyIPRejected  = errors.New("ip address belongs to trusted proxy")
	ErrAddressNotOwnedByDevice = errors.New("address is not owned by the device")
	ErrInvalidAPIKey           = errors.New("invalid api key")
	ErrInvalidGranularity      = errors.New("invalid granularity: must be 'hour' or 'day'")
)
