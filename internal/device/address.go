package device

import (
	"net/netip"
	"strconv"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
)

// Address represents an address row with its current enabled/disabled state and metadata.
type Address struct {
	ID        AddressID   `db:"id"`
	DeviceID  DeviceID    `db:"device_id"`
	IP        string      `db:"ip"`
	IsEnabled bool        `db:"is_enabled"`
	Source    EventSource `db:"source"`
	CreatedAt time.Time   `db:"created_at"`
	UpdatedAt time.Time   `db:"updated_at"`
}

// EventSource is an alias for the API-generated type, making openapi.yaml
// the single source of truth for valid values.
type EventSource = httpapi.AddressEventSource

const (
	EventSourceHeartbeat     = httpapi.Heartbeat
	EventSourceManual        = httpapi.Manual
	EventSourceExpiry        = httpapi.Expiry
	EventSourceLimitExceeded = httpapi.LimitExceeded
)

// CreateAddressParams holds only what is necessary to create an address.
type CreateAddressParams struct {
	DeviceID DeviceID
	IP       netip.Addr
}

func NewCreateAddressParams(deviceID DeviceID, ipAddress string, trustedProxy netip.Addr) (CreateAddressParams, error) {
	parsedIP, err := ParseAndValidateIP(ipAddress)
	if err != nil {
		return CreateAddressParams{}, err
	}
	if parsedIP.IsLoopback() || parsedIP.IsMulticast() || parsedIP.IsUnspecified() || parsedIP.IsLinkLocalUnicast() {
		return CreateAddressParams{}, ErrInvalidDeviceIP
	}
	if trustedProxy.IsValid() && trustedProxy.Compare(parsedIP) == 0 {
		return CreateAddressParams{}, ErrTrustedProxyIPRejected
	}

	return CreateAddressParams{
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

// IPEntry associates an enabled IP address with the device and address that last registered it.
// When multiple devices share the same IP, the most recently active device wins.
type IPEntry struct {
	IP        string    `db:"ip"`
	DeviceID  DeviceID  `db:"device_id"`
	AddressID AddressID `db:"address_id"`
}

// ParseAndValidateIP parses and validates that the given string is a valid IPv4 or IPv6 address.
// It ignores the port if present and only cares about the IP component.
// TODO: Make private once the address_test go through the NewCreateAddressParams
func ParseAndValidateIP(ipInput string) (netip.Addr, error) {
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
