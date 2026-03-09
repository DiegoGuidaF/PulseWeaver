package device

import (
	"net/netip"
	"strconv"
	"time"
)

// Address maps the current enabled/disabled state and metadata
// for an address, joining data from addresses and address_current_state.
type Address struct {
	ID        AddressID   `db:"id"`
	DeviceID  DeviceID    `db:"device_id"`
	IP        string      `db:"ip"`
	IsEnabled bool        `db:"is_enabled"`
	Source    EventSource `db:"source"`
	CreatedAt time.Time   `db:"created_at"`
	UpdatedAt time.Time   `db:"updated_at"`
}

type EventSource string

const (
	EventSourceHeartbeat EventSource = "heartbeat"
	EventSourceManual    EventSource = "manual"
	EventSourceExpiry    EventSource = "expiry"
)

// CreateAddressParams holds only what is necessary to create an address.
type CreateAddressParams struct {
	DeviceID DeviceID
	IP       netip.Addr
}

func NewCreateAddressParams(deviceID DeviceID, ipAddress string, trustedProxy netip.Addr) (*CreateAddressParams, error) {
	parsedIP, err := parseAndValidateIP(ipAddress)
	if err != nil {
		return nil, err
	}
	if parsedIP.IsLoopback() || parsedIP.IsMulticast() || parsedIP.IsUnspecified() || parsedIP.IsLinkLocalUnicast() {
		return nil, ErrInvalidDeviceIP
	}
	if trustedProxy.IsValid() && trustedProxy.Compare(parsedIP) == 0 {
		return nil, ErrTrustedProxyIPRejected
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
func parseAndValidateIP(ipInput string) (netip.Addr, error) {
	// Try to parse as IP without port
	if parsedIP, err := netip.ParseAddr(ipInput); err == nil {
		return parsedIP, nil
	}

	// If that fails, try to parse as IP with port
	if ap, err := netip.ParseAddrPort(ipInput); err == nil {
		ipAddr := ap.Addr()
		return ipAddr, nil
	}

	// If both fail, return error
	return netip.Addr{}, ErrInvalidIPFormat
}
