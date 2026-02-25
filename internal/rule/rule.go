package rule

import (
	"encoding/json"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
)

// RuleID represents the primary key of a row in the device_rules table.
type RuleID int64

// RuleType is a string identifier for a rule kind (e.g. "ip_auto_expiry").
type RuleType string

const (
	// RuleTypeIPAutoExpiry controls automatic expiry of IP addresses for a device.
	RuleTypeIPAutoExpiry RuleType = "ip_auto_expiry"
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

// RuleConfigIPAutoExpiryConfig is the typed configuration for the ip_auto_expiry rule.
// TTLSeconds is the number of seconds after which an enabled IP should expire.
type RuleConfigIPAutoExpiryConfig struct {
	TTLSeconds int `json:"ttl_seconds"`
}

// ParseIPAutoExpiryConfig parses the JSON config for an ip_auto_expiry rule
// and enforces basic invariants.
func parseRuleConfigIPAutoExpiry(raw json.RawMessage) (*RuleConfigIPAutoExpiryConfig, error) {
	if len(raw) == 0 {
		return nil, ErrInvalidRuleConfig
	}

	var cfg RuleConfigIPAutoExpiryConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, ErrInvalidRuleConfig
	}

	// Enforce a sane lower bound; the exact minimum is also validated at the API layer.
	if cfg.TTLSeconds <= 0 {
		return nil, ErrInvalidRuleConfig
	}

	return &cfg, nil
}
