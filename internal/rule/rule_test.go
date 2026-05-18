//go:build test

package rule

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/matryer/is"
)

func TestNewDeviceAddressLeaseConfig_ValidTTLs(t *testing.T) {
	is := is.New(t)

	tests := []struct {
		name string
		ttl  int
	}{
		{name: "zero_ttl_allowed", ttl: 0},
		{name: "positive_ttl", ttl: 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewDeviceAddressLeaseConfig(tt.ttl)
			is.NoErr(err)
			is.Equal(cfg.TTLSeconds, tt.ttl)
		})
	}
}

func TestNewDeviceAddressLeaseConfig_InvalidNegativeTTL(t *testing.T) {
	is := is.New(t)

	_, err := NewDeviceAddressLeaseConfig(-1)
	is.True(err != nil)
	is.Equal(err, ErrInvalidRuleConfig)
}

func TestParseDeviceAddressLeaseConfig_InvalidInputs(t *testing.T) {
	is := is.New(t)

	tests := []struct {
		name string
		raw  json.RawMessage
	}{
		{name: "empty_raw", raw: json.RawMessage{}},
		{name: "nil_raw", raw: nil},
		{name: "malformed_json", raw: json.RawMessage(`{"ttl_seconds":`)}, // invalid JSON
		{name: "negative_ttl", raw: json.RawMessage(`{"ttl_seconds":-5}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDeviceAddressLeaseConfig(tt.raw)
			is.True(err != nil)
			is.Equal(err, ErrInvalidRuleConfig)
		})
	}
}

func TestParseDeviceAddressLeaseConfig_ValidInputs(t *testing.T) {
	is := is.New(t)

	tests := []struct {
		name     string
		raw      json.RawMessage
		expected int
	}{
		{name: "zero_ttl", raw: json.RawMessage(`{"ttl_seconds":0}`), expected: 0},
		{name: "positive_ttl", raw: json.RawMessage(`{"ttl_seconds":3600}`), expected: 3600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseDeviceAddressLeaseConfig(tt.raw)
			is.NoErr(err)
			is.Equal(cfg.TTLSeconds, tt.expected)
		})
	}
}

func TestRule_ToDeviceAddressLeaseRule_InvalidType(t *testing.T) {
	is := is.New(t)

	rule := &Rule{
		ID:       ids.RuleID(1),
		DeviceID: ids.DeviceID(42),
		RuleType: RuleType("other_type"),
		Enabled:  true,
		Config:   json.RawMessage(`{"ttl_seconds":60}`),
	}

	result, err := rule.ToDeviceAddressLeaseRule()
	is.True(err != nil)
	is.True(errors.Is(err, ErrInvalidRuleConfig))
	is.True(result == nil)
}

func TestRule_ToDeviceAddressLeaseRule_InvalidConfig(t *testing.T) {
	is := is.New(t)

	rule := &Rule{
		ID:       ids.RuleID(1),
		DeviceID: ids.DeviceID(42),
		RuleType: RuleTypeDeviceAddressLease,
		Enabled:  true,
		Config:   json.RawMessage(`{"ttl_seconds":-10}`),
	}

	result, err := rule.ToDeviceAddressLeaseRule()
	is.True(err != nil)
	is.Equal(err, ErrInvalidRuleConfig)
	is.True(result == nil)
}

func TestRule_ToDeviceAddressLeaseRule_ValidRule(t *testing.T) {
	is := is.New(t)

	cfg := &DeviceAddressLeaseConfig{TTLSeconds: 120}
	configBytes, err := json.Marshal(cfg)
	is.NoErr(err)

	now := time.Now().UTC()
	rule := &Rule{
		ID:        ids.RuleID(1),
		DeviceID:  ids.DeviceID(42),
		RuleType:  RuleTypeDeviceAddressLease,
		Enabled:   true,
		Config:    json.RawMessage(configBytes),
		CreatedAt: now,
		UpdatedAt: now,
	}

	result, err := rule.ToDeviceAddressLeaseRule()
	is.NoErr(err)
	is.True(result != nil)
	is.Equal(result.ID, rule.ID)
	is.Equal(result.DeviceID, rule.DeviceID)
	is.Equal(result.Enabled, rule.Enabled)
	is.Equal(result.Config.TTLSeconds, cfg.TTLSeconds)
	is.Equal(result.CreatedAt, rule.CreatedAt)
	is.Equal(result.UpdatedAt, rule.UpdatedAt)
}

// MaxActiveAddressesConfig tests

func TestNewMaxActiveAddressesConfig_Valid(t *testing.T) {
	is := is.New(t)

	tests := []struct {
		name         string
		maxAddresses int
	}{
		{name: "min_value", maxAddresses: 1},
		{name: "typical_value", maxAddresses: 5},
		{name: "large_value", maxAddresses: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewMaxActiveAddressesConfig(tt.maxAddresses)
			is.NoErr(err)
			is.Equal(cfg.MaxAddresses, tt.maxAddresses)
		})
	}
}

func TestNewMaxActiveAddressesConfig_Invalid(t *testing.T) {
	is := is.New(t)

	tests := []struct {
		name         string
		maxAddresses int
	}{
		{name: "zero", maxAddresses: 0},
		{name: "negative", maxAddresses: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMaxActiveAddressesConfig(tt.maxAddresses)
			is.True(err != nil)
			is.True(errors.Is(err, ErrInvalidMaxAddresses))
		})
	}
}

func TestParseMaxActiveAddressesConfig_Valid(t *testing.T) {
	is := is.New(t)

	cfg, err := parseMaxActiveAddressesConfig(json.RawMessage(`{"max_addresses":3}`))
	is.NoErr(err)
	is.Equal(cfg.MaxAddresses, 3)
}

func TestParseMaxActiveAddressesConfig_Invalid(t *testing.T) {
	is := is.New(t)

	tests := []struct {
		name string
		raw  json.RawMessage
	}{
		{name: "empty", raw: json.RawMessage{}},
		{name: "nil", raw: nil},
		{name: "malformed_json", raw: json.RawMessage(`{"max_addresses":`)},
		{name: "zero_value", raw: json.RawMessage(`{"max_addresses":0}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseMaxActiveAddressesConfig(tt.raw)
			is.True(err != nil)
			is.True(errors.Is(err, ErrInvalidRuleConfig))
		})
	}
}

func TestRule_ToMaxActiveAddressesRule_WrongType(t *testing.T) {
	is := is.New(t)

	r := &Rule{
		ID:       ids.RuleID(1),
		DeviceID: ids.DeviceID(42),
		RuleType: RuleTypeDeviceAddressLease,
		Enabled:  true,
		Config:   json.RawMessage(`{"max_addresses":3}`),
	}

	result, err := r.ToMaxActiveAddressesRule()
	is.True(err != nil)
	is.True(errors.Is(err, ErrInvalidRuleConfig))
	is.True(result == nil)
}

func TestRule_ToMaxActiveAddressesRule_InvalidConfig(t *testing.T) {
	is := is.New(t)

	r := &Rule{
		ID:       ids.RuleID(1),
		DeviceID: ids.DeviceID(42),
		RuleType: RuleTypeMaxActiveAddresses,
		Enabled:  true,
		Config:   json.RawMessage(`{"max_addresses":0}`),
	}

	result, err := r.ToMaxActiveAddressesRule()
	is.True(err != nil)
	is.True(errors.Is(err, ErrInvalidRuleConfig))
	is.True(result == nil)
}

func TestRule_ToMaxActiveAddressesRule_Valid(t *testing.T) {
	is := is.New(t)

	now := time.Now().UTC()
	r := &Rule{
		ID:        ids.RuleID(7),
		DeviceID:  ids.DeviceID(99),
		RuleType:  RuleTypeMaxActiveAddresses,
		Enabled:   true,
		Config:    json.RawMessage(`{"max_addresses":5}`),
		CreatedAt: now,
		UpdatedAt: now,
	}

	result, err := r.ToMaxActiveAddressesRule()
	is.NoErr(err)
	is.True(result != nil)
	is.Equal(result.ID, r.ID)
	is.Equal(result.DeviceID, r.DeviceID)
	is.Equal(result.Enabled, r.Enabled)
	is.Equal(result.Config.MaxAddresses, 5)
	is.Equal(result.CreatedAt, r.CreatedAt)
	is.Equal(result.UpdatedAt, r.UpdatedAt)
}
