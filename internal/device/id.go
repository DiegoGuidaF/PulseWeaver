package device

import (
	"fmt"
	"strconv"
)

type DeviceID int64

func NewDeviceID(idString string) (DeviceID, error) {
	return parseIdFromString[DeviceID](idString, "DeviceID")
}

func (id DeviceID) Int64() int64 {
	return int64(id)
}

func (id DeviceID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

type DeviceIpID int64

func NewDeviceIPID(idString string) (DeviceIpID, error) {
	return parseIdFromString[DeviceIpID](idString, "DeviceIpID")
}

func (id DeviceIpID) Int64() int64 {
	return int64(id)
}

func (id DeviceIpID) String() string {
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
