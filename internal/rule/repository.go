package rule

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
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
	rule := &Rule{}

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

// UpsertRule creates or updates a rule for a device, identified by (device_id, rule_type).
func (r *Repository) UpsertRule(ctx context.Context, rule *Rule) (*Rule, error) {
	now := time.Now().UTC()
	const query = `
		INSERT INTO device_rules (device_id, rule_type, enabled, config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(device_id, rule_type) DO UPDATE SET
			enabled   = excluded.enabled,
			config    = excluded.config,
			updated_at = excluded.updated_at
		RETURNING id, device_id, rule_type, enabled, config, created_at, updated_at
	`

	if err := r.db.GetContext(ctx, rule, query,
		rule.DeviceID,
		rule.RuleType,
		rule.Enabled,
		rule.Config,
		now,
		now,
	); err != nil {
		return nil, fmt.Errorf("upsert rule: %w", err)
	}

	return rule, nil
}

// DisableRule Sets enabled false for rule identified by (device_id, rule_type).
func (r *Repository) DisableRule(ctx context.Context, deviceID device.DeviceID, ruleType RuleType) (*Rule, error) {
	rule := &Rule{}

	const query = `
		UPDATE device_rules SET enabled = FALSE, updated_at = ?
		WHERE device_id = ? AND rule_type = ?
	`

	err := r.db.GetContext(ctx, rule, query, time.Now().UTC(), deviceID, ruleType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRuleNotFound
		}
		return nil, fmt.Errorf("delete rule: %w", err)
	}

	return rule, nil
}
