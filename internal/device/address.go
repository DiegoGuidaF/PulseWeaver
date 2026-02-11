package device

import (
	"net/netip"
	"strconv"
	"time"
)

type Address struct {
	ID        AddressID `db:"id"`
	DeviceId  DeviceID  `db:"device_id"`
	IP        string    `db:"ip"`
	CreatedAt time.Time `db:"created_at"`
}

func NewAddress(deviceId DeviceID, ipAddress string) (*Address, error) {
	parsedIp, err := parseAndValidateIP(ipAddress)
	if err != nil {
		return nil, err
	}

	return &Address{
		DeviceId:  deviceId,
		IP:        parsedIp,
		CreatedAt: time.Now().UTC(),
	}, nil
}

type AddressID int64

func (id AddressID) Int64() int64 {
	return int64(id)
}

func (id AddressID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// parseAndValidateIP parses and validates that the given string is a valid IPv4 or IPv6 address.
// It ignores the port if present and only cares about the IP component.
func parseAndValidateIP(ipInput string) (string, error) {
	// Try to parse as IP without port
	if ip, err := netip.ParseAddr(ipInput); err == nil {
		ipStr := ip.String()
		return ipStr, nil
	}

	// If that fails, try to parse as IP with port
	if ap, err := netip.ParseAddrPort(ipInput); err == nil {
		ipStr := ap.Addr().String()
		return ipStr, nil
	}

	// If both fail, return error
	return "", ErrInvalidIPFormat
}
