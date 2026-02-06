package device

import (
	"fmt"
	"strconv"
)

type DeviceId int64

func NewDeviceID(idString string) (DeviceId, error) {
	return parseIdFromString[DeviceId](idString, "DeviceId")
}

func (id DeviceId) Int64() int64 {
	return int64(id)
}

func (id DeviceId) String() string {
	return strconv.FormatInt(int64(id), 10)
}

type AddressId int64

func NewDeviceIPID(idString string) (AddressId, error) {
	return parseIdFromString[AddressId](idString, "AddressId")
}

func (id AddressId) Int64() int64 {
	return int64(id)
}

func (id AddressId) String() string {
	return strconv.FormatInt(int64(id), 10)
}

func parseIdFromString[T ~int64](s string, name string) (T, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	if id <= 0 {
		return 0, fmt.Errorf("%s must be positive, got %d", name, id)
	}
	return T(id), nil
}
