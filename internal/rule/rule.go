package rule

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// RuleType is a string identifier for a rule kind (e.g. "device_lease").
type RuleType string

const (
	// RuleTypeDeviceAddressLease controls automatic expiry of IP addresses for a device.
	RuleTypeDeviceAddressLease RuleType = "device_lease"

	// RuleTypeMaxActiveAddresses limits the number of simultaneously enabled IP addresses per device.
	RuleTypeMaxActiveAddresses RuleType = "max_active_addresses"
)

// Rule maps to a row in the device_rules table.
type Rule struct {
	ID        ids.RuleID      `db:"id"`
	DeviceID  ids.DeviceID    `db:"device_id"`
	RuleType  RuleType        `db:"rule_type"`
	Enabled   bool            `db:"enabled"`
	Config    json.RawMessage `db:"config"`
	CreatedAt time.Time       `db:"created_at"`
	UpdatedAt time.Time       `db:"updated_at"`
}

// DeviceAddressLeaseRule represents the domain model for the device lease rule.
type DeviceAddressLeaseRule struct {
	ID        ids.RuleID
	DeviceID  ids.DeviceID
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
		Config:    config,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}, nil
}

// parseDeviceAddressLeaseConfig parses the JSON config for a device lease rule
func parseDeviceAddressLeaseConfig(raw json.RawMessage) (DeviceAddressLeaseConfig, error) {
	if len(raw) == 0 {
		return DeviceAddressLeaseConfig{}, ErrInvalidRuleConfig
	}

	var cfg DeviceAddressLeaseConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return DeviceAddressLeaseConfig{}, ErrInvalidRuleConfig
	}

	if cfg.TTLSeconds < 0 {
		return DeviceAddressLeaseConfig{}, ErrInvalidRuleConfig
	}

	return cfg, nil
}

// NewDeviceAddressLeaseConfig Creates an addess lease config and validates the parameters
func NewDeviceAddressLeaseConfig(addressTTLSeconds int) (DeviceAddressLeaseConfig, error) {
	if addressTTLSeconds < 0 {
		return DeviceAddressLeaseConfig{}, ErrInvalidRuleConfig
	}
	return DeviceAddressLeaseConfig{TTLSeconds: addressTTLSeconds}, nil
}

// MaxActiveAddressesRule represents the domain model for the max active addresses rule.
type MaxActiveAddressesRule struct {
	ID        ids.RuleID
	DeviceID  ids.DeviceID
	Enabled   bool
	Config    MaxActiveAddressesConfig
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MaxActiveAddressesConfig holds configuration for the max active addresses rule.
type MaxActiveAddressesConfig struct {
	MaxAddresses int `json:"max_addresses"`
}

// NewMaxActiveAddressesConfig validates and creates a MaxActiveAddressesConfig.
func NewMaxActiveAddressesConfig(maxAddresses int) (MaxActiveAddressesConfig, error) {
	if maxAddresses < 1 {
		return MaxActiveAddressesConfig{}, ErrInvalidMaxAddresses
	}
	return MaxActiveAddressesConfig{MaxAddresses: maxAddresses}, nil
}

// parseMaxActiveAddressesConfig parses the JSON config for a max active addresses rule.
func parseMaxActiveAddressesConfig(raw json.RawMessage) (MaxActiveAddressesConfig, error) {
	if len(raw) == 0 {
		return MaxActiveAddressesConfig{}, ErrInvalidRuleConfig
	}
	var cfg MaxActiveAddressesConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return MaxActiveAddressesConfig{}, ErrInvalidRuleConfig
	}
	if cfg.MaxAddresses < 1 {
		return MaxActiveAddressesConfig{}, ErrInvalidRuleConfig
	}
	return cfg, nil
}

// ToMaxActiveAddressesRule converts the raw Rule row into a MaxActiveAddressesRule.
func (r *Rule) ToMaxActiveAddressesRule() (*MaxActiveAddressesRule, error) {
	if r.RuleType != RuleTypeMaxActiveAddresses {
		return nil, fmt.Errorf("%w: invalid rule type %s", ErrInvalidRuleConfig, r.RuleType)
	}
	config, err := parseMaxActiveAddressesConfig(r.Config)
	if err != nil {
		return nil, err
	}
	return &MaxActiveAddressesRule{
		ID:        r.ID,
		DeviceID:  r.DeviceID,
		Enabled:   r.Enabled,
		Config:    config,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}, nil
}
