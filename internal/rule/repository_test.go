//go:build test

package rule

import (
	"context"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

func setupRuleTestDB(t *testing.T) *Repository {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	return NewRepository(db.DB())
}

func createTestDevice(t *testing.T, repo *Repository, ctx context.Context, name string) *device.Device {
	t.Helper()
	devRepo := device.NewRepository(repo.rootDB)
	params, _, err := device.NewCreateDeviceParams(name)
	if err != nil {
		t.Fatalf("create device params: %v", err)
	}
	dev, err := devRepo.CreateDevice(ctx, params)
	if err != nil {
		t.Fatalf("create device: %v", err)
	}
	return dev
}

func TestRepository_GetRuleByDeviceAndType(t *testing.T) {
	is := is.New(t)
	repo := setupRuleTestDB(t)
	ctx := context.Background()

	t.Run("not_found", func(t *testing.T) {
		is := is.New(t)
		rule, err := repo.GetRuleByDeviceAndType(ctx, device.DeviceID(99999), RuleTypeDeviceAddressLease)
		is.True(err != nil)
		is.Equal(err, ErrRuleNotFound)
		is.True(rule == nil)
	})

	t.Run("returns_rule_after_enable", func(t *testing.T) {
		is := is.New(t)
		dev := createTestDevice(t, repo, ctx, "rule-device")
		cfg, err := NewDeviceAddressLeaseConfig(60)
		is.NoErr(err)
		_, err = repo.EnableDeviceAddressLeaseRuleConfig(ctx, dev.ID, cfg)
		is.NoErr(err)

		rule, err := repo.GetRuleByDeviceAndType(ctx, dev.ID, RuleTypeDeviceAddressLease)
		is.NoErr(err)
		is.True(rule != nil)
		is.Equal(rule.DeviceID, dev.ID)
		is.Equal(rule.RuleType, RuleTypeDeviceAddressLease)
		is.True(rule.Enabled)
	})
}

func TestRepository_EnableDeviceAddressLeaseRuleConfig(t *testing.T) {
	is := is.New(t)
	repo := setupRuleTestDB(t)
	ctx := context.Background()

	t.Run("creates_rule", func(t *testing.T) {
		is := is.New(t)
		dev := createTestDevice(t, repo, ctx, "enable-device")
		cfg, err := NewDeviceAddressLeaseConfig(300)
		is.NoErr(err)

		rule, err := repo.EnableDeviceAddressLeaseRuleConfig(ctx, dev.ID, cfg)
		is.NoErr(err)
		is.True(rule != nil)
		is.Equal(rule.DeviceID, dev.ID)
		is.Equal(rule.RuleType, RuleTypeDeviceAddressLease)
		is.True(rule.Enabled)
		is.True(len(rule.Config) > 0)
	})

	t.Run("non_existent_device_returns_device_not_found", func(t *testing.T) {
		is := is.New(t)
		cfg, err := NewDeviceAddressLeaseConfig(90)
		is.NoErr(err)

		rule, err := repo.EnableDeviceAddressLeaseRuleConfig(ctx, device.DeviceID(99999), cfg)
		is.True(err != nil)
		is.Equal(err, device.ErrDeviceNotFound)
		is.True(rule == nil)
	})
}

func TestRepository_DisableRule(t *testing.T) {
	is := is.New(t)
	repo := setupRuleTestDB(t)
	ctx := context.Background()

	t.Run("not_found", func(t *testing.T) {
		is := is.New(t)
		rule, err := repo.DisableRule(ctx, device.DeviceID(12345), RuleTypeDeviceAddressLease)
		is.True(err != nil)
		is.Equal(err, ErrRuleNotFound)
		is.True(rule == nil)
	})

	t.Run("disables_existing_rule", func(t *testing.T) {
		is := is.New(t)
		dev := createTestDevice(t, repo, ctx, "disable-device")
		cfg, err := NewDeviceAddressLeaseConfig(180)
		is.NoErr(err)
		_, err = repo.EnableDeviceAddressLeaseRuleConfig(ctx, dev.ID, cfg)
		is.NoErr(err)

		rule, err := repo.DisableRule(ctx, dev.ID, RuleTypeDeviceAddressLease)
		is.NoErr(err)
		is.True(rule != nil)
		is.True(!rule.Enabled)
	})
}
