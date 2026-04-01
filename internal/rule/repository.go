package rule

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/jmoiron/sqlx"
)

type dBInterface interface {
	sqlx.ExtContext
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// Repository provides SQL-backed persistence for device rules.
type Repository struct {
	db     dBInterface
	rootDB *sqlx.DB
}

// NewRepository creates a new Repository backed by the given sqlx.DB.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		db:     db,
		rootDB: db,
	}
}

// GetRuleByDeviceAndType returns a single rule for the given device and type.
func (r *Repository) GetRuleByDeviceAndType(ctx context.Context, deviceID device.DeviceID, ruleType RuleType) (*Rule, error) {
	rule := new(Rule)

	const query = `
		SELECT *
		FROM device_rules
		WHERE device_id = ? AND rule_type = ?
	`

	if err := r.db.GetContext(ctx, rule, query, deviceID, ruleType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRuleNotFound
		}
		return nil, fmt.Errorf("get rule by device and type: %w", err)
	}

	return rule, nil
}

// DisableRule sets enabled=false for the rule identified by (device_id, rule_type).
func (r *Repository) DisableRule(ctx context.Context, deviceID device.DeviceID, ruleType RuleType) (*Rule, error) {
	rule := new(Rule)

	const query = `
		UPDATE device_rules 
		SET enabled = FALSE, updated_at = ?
		WHERE device_id = ? AND rule_type = ?
		RETURNING *
	`

	if err := r.db.GetContext(ctx, rule, query, time.Now().UTC(), deviceID, ruleType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRuleNotFound
		}
		return nil, mapRepositoryError(err, "disable rule")
	}

	return rule, nil
}

// EnableDeviceAddressLeaseRuleConfig creates or updates the device lease rule for a device
// using the structured params. It is responsible for mapping the config into
// the JSON shape stored in the database.
func (r *Repository) EnableDeviceAddressLeaseRuleConfig(ctx context.Context, deviceID device.DeviceID, config DeviceAddressLeaseConfig) (*Rule, error) {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshal rule config: %w", err)
	}

	rule := new(Rule)
	const query = `
		INSERT INTO device_rules (device_id, rule_type, enabled, config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(device_id, rule_type) DO UPDATE SET
			enabled = excluded.enabled,
			config = excluded.config,
			updated_at = excluded.updated_at
		RETURNING *
	`

	now := time.Now().UTC()
	if err := r.db.GetContext(ctx, rule, query,
		deviceID,
		RuleTypeDeviceAddressLease,
		true,
		configBytes,
		now,
		now,
	); err != nil {
		return nil, mapRepositoryError(err, "enable device lease rule")
	}

	return rule, nil
}

// EnableMaxActiveAddressesRuleConfig creates or updates the max active addresses rule for a device.
func (r *Repository) EnableMaxActiveAddressesRuleConfig(ctx context.Context, deviceID device.DeviceID, config MaxActiveAddressesConfig) (*Rule, error) {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshal rule config: %w", err)
	}

	rule := new(Rule)
	const query = `
		INSERT INTO device_rules (device_id, rule_type, enabled, config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(device_id, rule_type) DO UPDATE SET
			enabled = excluded.enabled,
			config = excluded.config,
			updated_at = excluded.updated_at
		RETURNING *
	`

	now := time.Now().UTC()
	if err := r.db.GetContext(ctx, rule, query,
		deviceID,
		RuleTypeMaxActiveAddresses,
		true,
		configBytes,
		now,
		now,
	); err != nil {
		return nil, mapRepositoryError(err, "enable max active addresses rule")
	}

	return rule, nil
}

// mapRuleForeignKeyDeviceError maps a SQLite foreign key constraint failure on device_rules.device_id
// to the domain-level device.ErrDeviceNotFound error.
func mapRuleForeignKeyDeviceError(err error) (error, bool) {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "foreign key constraint failed") {
		return device.ErrDeviceNotFound, true
	}
	return nil, false
}

// mapRepositoryError applies repository-level error mapping and wraps with context.
func mapRepositoryError(err error, operation string) error {
	if mappedErr, ok := mapRuleForeignKeyDeviceError(err); ok {
		return mappedErr
	}
	return fmt.Errorf("%s: %w", operation, err)
}
