//go:build test

package rule_test

import (
	"context"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/jmoiron/sqlx"
	"github.com/matryer/is"
)

func setupRuleTestDB(t *testing.T) (*rule.Repository, *sqlx.DB) {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	sqlDB := db.DB()
	return rule.NewRepository(sqlDB), sqlDB
}

func insertDevice(t *testing.T, db *sqlx.DB, ctx context.Context, name string) *device.Device {
	t.Helper()
	devRepo := device.NewRepository(db)
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

func TestRepository_GetRuleByDeviceAndType_NotFound(t *testing.T) {
	is := is.New(t)
	repo, _ := setupRuleTestDB(t)
	ctx := context.Background()

	r, err := repo.GetRuleByDeviceAndType(ctx, device.DeviceID(99999), rule.RuleTypeDeviceAddressLease)

	is.True(err != nil)
	is.Equal(err, rule.ErrRuleNotFound)
	is.True(r == nil)
}

func TestRepository_GetRuleByDeviceAndType_ReturnsRuleAfterEnable(t *testing.T) {
	is := is.New(t)
	repo, db := setupRuleTestDB(t)
	ctx := context.Background()
	dev := insertDevice(t, db, ctx, "rule-device")
	cfg, err := rule.NewDeviceAddressLeaseConfig(60)
	is.NoErr(err)
	_, err = repo.EnableDeviceAddressLeaseRuleConfig(ctx, dev.ID, cfg)
	is.NoErr(err)

	r, err := repo.GetRuleByDeviceAndType(ctx, dev.ID, rule.RuleTypeDeviceAddressLease)

	is.NoErr(err)
	is.True(r != nil)
	is.Equal(r.DeviceID, dev.ID)
	is.Equal(r.RuleType, rule.RuleTypeDeviceAddressLease)
	is.True(r.Enabled)
}

func TestRepository_EnableDeviceAddressLeaseRuleConfig_CreatesRule(t *testing.T) {
	is := is.New(t)
	repo, db := setupRuleTestDB(t)
	ctx := context.Background()
	dev := insertDevice(t, db, ctx, "enable-device")
	cfg, err := rule.NewDeviceAddressLeaseConfig(300)
	is.NoErr(err)

	r, err := repo.EnableDeviceAddressLeaseRuleConfig(ctx, dev.ID, cfg)

	is.NoErr(err)
	is.True(r != nil)
	is.Equal(r.DeviceID, dev.ID)
	is.Equal(r.RuleType, rule.RuleTypeDeviceAddressLease)
	is.True(r.Enabled)
	is.True(len(r.Config) > 0)
}

func TestRepository_EnableDeviceAddressLeaseRuleConfig_NonExistentDevice(t *testing.T) {
	is := is.New(t)
	repo, _ := setupRuleTestDB(t)
	ctx := context.Background()
	cfg, err := rule.NewDeviceAddressLeaseConfig(90)
	is.NoErr(err)

	r, err := repo.EnableDeviceAddressLeaseRuleConfig(ctx, device.DeviceID(99999), cfg)

	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
	is.True(r == nil)
}

func TestRepository_DisableRule_NotFound(t *testing.T) {
	is := is.New(t)
	repo, _ := setupRuleTestDB(t)
	ctx := context.Background()

	r, err := repo.DisableRule(ctx, device.DeviceID(12345), rule.RuleTypeDeviceAddressLease)

	is.True(err != nil)
	is.Equal(err, rule.ErrRuleNotFound)
	is.True(r == nil)
}

func TestRepository_DisableRule_DisablesExistingRule(t *testing.T) {
	is := is.New(t)
	repo, db := setupRuleTestDB(t)
	ctx := context.Background()
	dev := insertDevice(t, db, ctx, "disable-device")
	cfg, err := rule.NewDeviceAddressLeaseConfig(180)
	is.NoErr(err)
	_, err = repo.EnableDeviceAddressLeaseRuleConfig(ctx, dev.ID, cfg)
	is.NoErr(err)

	r, err := repo.DisableRule(ctx, dev.ID, rule.RuleTypeDeviceAddressLease)

	is.NoErr(err)
	is.True(r != nil)
	is.True(!r.Enabled)
}
