//go:build test

package rule_test

import (
	"context"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

func setupRuleTestDB(t *testing.T) (*rule.Repository, *database.DB) {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	sqlDB := db.DB()
	return rule.NewRepository(sqlDB), sqlDB
}

func ensureTestOwner(t *testing.T, db *database.DB, ctx context.Context) auth.UserID {
	t.Helper()
	_, _ = db.ExecContext(ctx, `INSERT OR IGNORE INTO users (username, display_name, password_hash, role) VALUES ('testowner', 'Test Owner', 'x', 'admin')`)
	var id auth.UserID
	if err := db.QueryRowxContext(ctx, `SELECT id FROM users WHERE username = 'testowner'`).Scan(&id); err != nil {
		t.Fatalf("ensureTestOwner: %v", err)
	}
	return id
}

func insertDevice(t *testing.T, db *database.DB, ctx context.Context, name string) *device.Device {
	t.Helper()
	ownerID := ensureTestOwner(t, db, ctx)
	devRepo := device.NewRepository(db)
	dev, err := devRepo.CreateDevice(ctx, device.CreateDeviceParams{Name: name, OwnerID: ownerID})
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

func TestRepository_EnableMaxActiveAddressesRuleConfig_Creates(t *testing.T) {
	is := is.New(t)
	repo, db := setupRuleTestDB(t)
	ctx := context.Background()
	dev := insertDevice(t, db, ctx, "max-addr-create-device")
	cfg, err := rule.NewMaxActiveAddressesConfig(3)
	is.NoErr(err)

	r, err := repo.EnableMaxActiveAddressesRuleConfig(ctx, dev.ID, cfg)

	is.NoErr(err)
	is.True(r != nil)
	is.Equal(r.DeviceID, dev.ID)
	is.Equal(r.RuleType, rule.RuleTypeMaxActiveAddresses)
	is.True(r.Enabled)
	is.True(len(r.Config) > 0)
}

func TestRepository_EnableMaxActiveAddressesRuleConfig_Upsert(t *testing.T) {
	is := is.New(t)
	repo, db := setupRuleTestDB(t)
	ctx := context.Background()
	dev := insertDevice(t, db, ctx, "max-addr-upsert-device")

	cfg1, _ := rule.NewMaxActiveAddressesConfig(3)
	r1, err := repo.EnableMaxActiveAddressesRuleConfig(ctx, dev.ID, cfg1)
	is.NoErr(err)
	is.True(r1 != nil)

	cfg2, _ := rule.NewMaxActiveAddressesConfig(5)
	r2, err := repo.EnableMaxActiveAddressesRuleConfig(ctx, dev.ID, cfg2)
	is.NoErr(err)
	is.True(r2 != nil)
	is.Equal(r1.ID, r2.ID)

	// Verify only one rule exists
	fetched, err := repo.GetRuleByDeviceAndType(ctx, dev.ID, rule.RuleTypeMaxActiveAddresses)
	is.NoErr(err)
	maxRule, err := fetched.ToMaxActiveAddressesRule()
	is.NoErr(err)
	is.Equal(maxRule.Config.MaxAddresses, 5)
}

func TestRepository_EnableMaxActiveAddressesRuleConfig_NonExistentDevice(t *testing.T) {
	is := is.New(t)
	repo, _ := setupRuleTestDB(t)
	ctx := context.Background()
	cfg, _ := rule.NewMaxActiveAddressesConfig(3)

	r, err := repo.EnableMaxActiveAddressesRuleConfig(ctx, device.DeviceID(99999), cfg)

	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
	is.True(r == nil)
}
