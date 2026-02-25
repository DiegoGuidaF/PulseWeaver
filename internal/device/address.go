package device

import (
	"net/netip"
	"strconv"
	"time"
)

// Address maps the current enabled/disabled state and metadata
// for an address, joining data from addresses and address_current_state.
type Address struct {
	ID        AddressID    `db:"id"`
	DeviceID  DeviceID     `db:"device_id"`
	IP        string       `db:"ip"`
	Status    bool         `db:"is_enabled"`
	Source    StatusSource `db:"source"`
	ExpiresAt *time.Time   `db:"expires_at"`
	CreatedAt time.Time    `db:"created_at"`
	UpdatedAt time.Time    `db:"updated_at"`
}

type StatusSource string

const (
	StatusSourceHeartbeat StatusSource = "heartbeat"
	StatusSourceManual    StatusSource = "manual"
	StatusSourceExpiry    StatusSource = "expiry"
)

// CreateAddressParams holds only what is necessary to create an address.
type CreateAddressParams struct {
	DeviceID DeviceID
	IP       string
}

func NewCreateAddressParams(deviceID DeviceID, ipAddress string) (*CreateAddressParams, error) {
	parsedIP, err := parseAndValidateIP(ipAddress)
	if err != nil {
		return nil, err
	}

	return &CreateAddressParams{
		DeviceID: deviceID,
		IP:       parsedIP,
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
