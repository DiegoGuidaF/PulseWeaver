package device

import (
	"net/netip"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// Address represents an address row with its current enabled/disabled state and metadata.
type Address struct {
	ID        ids.AddressID `db:"id"`
	DeviceID  ids.DeviceID  `db:"device_id"`
	IP        string        `db:"ip"`
	IsEnabled bool          `db:"is_enabled"`
	Source    EventSource   `db:"source"`
	CreatedAt time.Time     `db:"created_at"`
	UpdatedAt time.Time     `db:"updated_at"`
}

// EventSource is an alias for the API-generated type, making openapi.yaml
// the single source of truth for valid values. It records which subsystem
// wrote an address event; EventTrigger records what set it off.
type EventSource = httpapi.AddressEventSource

const (
	EventSourceHeartbeat     = httpapi.Heartbeat
	EventSourceWebUI         = httpapi.WebUi
	EventSourceExpiry        = httpapi.Expiry
	EventSourceLimitExceeded = httpapi.LimitExceeded
)

// EventTrigger is an alias for the API-generated type, making openapi.yaml
// the single source of truth for valid values. It is orthogonal to
// EventSource: the same subsystem writes events for different reasons, and
// the pairing is always a caller decision — a heartbeat can be scheduled or
// user-pressed, so neither axis may be derived from the other.
type EventTrigger = httpapi.AddressEventTrigger

const (
	EventTriggerUser          = httpapi.AddressEventTriggerUser
	EventTriggerSchedule      = httpapi.AddressEventTriggerSchedule
	EventTriggerNetworkChange = httpapi.AddressEventTriggerNetworkChange
	EventTriggerSystem        = httpapi.AddressEventTriggerSystem
)

// ParseEventTrigger maps a caller-supplied trigger onto a value safe to store.
// It takes a raw string rather than the enum because the heartbeat endpoints
// deliberately declare the parameter untyped: a value the request validator
// would reject costs the device its authorization, and a heartbeat must never
// fail over its own annotation. So an unrecognised value degrades here, as
// does EventTriggerSystem, which is server-set and not claimable over the
// wire. The bool reports whether v was recognised, so the caller can warn
// without deciding the fallback itself. An absent parameter is the caller's
// own case to handle: it is an ordinary older-client heartbeat, not something
// to warn about.
func ParseEventTrigger(v string) (EventTrigger, bool) {
	trigger := EventTrigger(v)
	if !trigger.Valid() || trigger == EventTriggerSystem {
		return EventTriggerSchedule, false
	}
	return trigger, true
}

// CreateAddressParams holds only what is necessary to create an address.
type CreateAddressParams struct {
	DeviceID ids.DeviceID
	IP       netip.Addr
}

func NewCreateAddressParams(deviceID ids.DeviceID, ipAddress string, trustedProxy netip.Addr) (CreateAddressParams, error) {
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

// IPEntry associates an enabled IP address with the device, address, and owning user.
// All enabled rows are returned (multiple per IP when devices share an address);
// the policy layer merges them with deny-wins intersection.
type IPEntry struct {
	IP        string        `db:"ip"`
	DeviceID  ids.DeviceID  `db:"device_id"`
	AddressID ids.AddressID `db:"address_id"`
	UserID    ids.UserID    `db:"user_id"`
}

// ParseAndValidateIP parses and validates that the given string is a valid IPv4 or IPv6 address.
// It ignores the port if present and only cares about the IP component.
// The result is always canonical: an IPv4-mapped IPv6 address (::ffff:a.b.c.d) is
// unmapped to its plain IPv4 form so stored addresses never depend on representation.
// TODO: Make private once the address_test go through the NewCreateAddressParams
func ParseAndValidateIP(ipInput string) (netip.Addr, error) {
	if parsedIP, err := netip.ParseAddr(ipInput); err == nil {
		return parsedIP.Unmap(), nil
	}
	if ap, err := netip.ParseAddrPort(ipInput); err == nil {
		return ap.Addr().Unmap(), nil
	}
	return netip.Addr{}, ErrInvalidIPFormat
}
