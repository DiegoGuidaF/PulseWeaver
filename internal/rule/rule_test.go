//go:build test

package rule

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
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
			is.True(cfg != nil)
			is.Equal(cfg.TTLSeconds, tt.ttl)
		})
	}
}

func TestNewDeviceAddressLeaseConfig_InvalidNegativeTTL(t *testing.T) {
	is := is.New(t)

	cfg, err := NewDeviceAddressLeaseConfig(-1)
	is.True(err != nil)
	is.Equal(err, ErrInvalidRuleConfig)
	is.True(cfg == nil)
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
			cfg, err := parseDeviceAddressLeaseConfig(tt.raw)
			is.True(err != nil)
			is.Equal(err, ErrInvalidRuleConfig)
			is.True(cfg == nil)
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
			is.True(cfg != nil)
			is.Equal(cfg.TTLSeconds, tt.expected)
		})
	}
}

func TestRule_ToDeviceAddressLeaseRule_InvalidType(t *testing.T) {
	is := is.New(t)

	rule := &Rule{
		ID:       RuleID(1),
		DeviceID: device.DeviceID(42),
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
		ID:       RuleID(1),
		DeviceID: device.DeviceID(42),
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
		ID:        RuleID(1),
		DeviceID:  device.DeviceID(42),
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
