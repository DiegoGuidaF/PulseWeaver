package rule

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
)

// RuleID represents the primary key of a row in the device_rules table.
type RuleID int64

// RuleType is a string identifier for a rule kind (e.g. "device_lease").
type RuleType string

const (
	// RuleTypeDeviceAddressLease controls automatic expiry of IP addresses for a device.
	RuleTypeDeviceAddressLease RuleType = "device_lease"
)

// Rule maps to a row in the device_rules table.
type Rule struct {
	ID        RuleID          `db:"id"`
	DeviceID  device.DeviceID `db:"device_id"`
	RuleType  RuleType        `db:"rule_type"`
	Enabled   bool            `db:"enabled"`
	Config    json.RawMessage `db:"config"`
	CreatedAt time.Time       `db:"created_at"`
	UpdatedAt time.Time       `db:"updated_at"`
}

// DeviceAddressLeaseRule represents the domain model for the device lease rule.
type DeviceAddressLeaseRule struct {
	ID        RuleID
	DeviceID  device.DeviceID
	Enabled   bool
	Config    DeviceAddressLeaseConfig
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DeviceAddressLeaseConfig holds configuration values for the device lease rule.
type DeviceAddressLeaseConfig struct {
	TTLSeconds int `json:"ttl_seconds"`
}

// ToDeviceAddressLeaseRule converts the raw Rule row into a DeviceAddressLeaseRule by parsing
// and validating the JSON config.
func (r *Rule) ToDeviceAddressLeaseRule() (*DeviceAddressLeaseRule, error) {
	if r.RuleType != RuleTypeDeviceAddressLease {
		return nil, fmt.Errorf("%w: invalid rule type %s", ErrInvalidRuleConfig, r.RuleType)
	}
	config, err := parseDeviceAddressLeaseConfig(r.Config)
	if err != nil {
		return nil, err
	}

	return &DeviceAddressLeaseRule{
		ID:        r.ID,
		DeviceID:  r.DeviceID,
		Enabled:   r.Enabled,
		Config:    *config,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}, nil
}

// parseDeviceAddressLeaseConfig parses the JSON config for a device lease rule
func parseDeviceAddressLeaseConfig(raw json.RawMessage) (*DeviceAddressLeaseConfig, error) {
	if len(raw) == 0 {
		return nil, ErrInvalidRuleConfig
	}

	var cfg DeviceAddressLeaseConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, ErrInvalidRuleConfig
	}

	if cfg.TTLSeconds < 0 {
		return nil, ErrInvalidRuleConfig
	}

	return &cfg, nil
}

// NewDeviceAddressLeaseConfig Creates an addess lease config and validates the parameters
func NewDeviceAddressLeaseConfig(addressTTLSeconds int) (*DeviceAddressLeaseConfig, error) {
	if addressTTLSeconds < 0 {
		return nil, ErrInvalidRuleConfig
	}
	return &DeviceAddressLeaseConfig{
		TTLSeconds: addressTTLSeconds,
	}, nil
}
