package device

import "errors"

var (
	ErrDeviceNotFound          = errors.New("device not found")
	ErrDeviceDisabled          = errors.New("device is disabled")
	ErrOwnerNotFound           = errors.New("device owner not found")
	ErrNoAPIKey                = errors.New("device has no API key")
	ErrDuplicateDeviceName     = errors.New("device name already in use")
	ErrAddressNotFound         = errors.New("device address not found")
	ErrInvalidIPFormat         = errors.New("invalid IP address format")
	ErrInvalidDeviceIP         = errors.New("ip address is not valid for device registration")
	ErrTrustedProxyIPRejected  = errors.New("ip address belongs to trusted proxy")
	ErrAddressNotOwnedByDevice = errors.New("address is not owned by the device")
	ErrInvalidAPIKey           = errors.New("invalid api key")
	ErrInvalidDeviceName       = errors.New("device name must be between 1 and 50 characters")
	ErrDescriptionTooLong      = errors.New("description must be 200 characters or fewer")
	ErrIconTooLong             = errors.New("icon must be 80 characters or fewer")
)
