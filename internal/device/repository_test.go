package device

import (
	"context"
	"fmt"
	"testing"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"github.com/matryer/is"
)

func setupTestDB(t *testing.T) *Repository {
	t.Helper()

	conf := config.ConfDB{
		Dsn:   fmt.Sprintf("file:%s?mode=memory&_loc=auto", t.Name()),
		Debug: false,
	}

	db, err := database.NewSQLite(conf)
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return NewRepository(db.DB())
}

func TestRepository_CreateDevice(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)
	is.Equal(device.Name, "test-device")
	is.True(!device.CreatedAt.IsZero())
}

func TestRepository_GetDevices_Empty(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	devices, err := repo.GetDevices(ctx)
	is.NoErr(err)
	is.Equal(len(devices), 0) // Should be empty
}

func TestRepository_GetDevices_Multiple(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create test data
	_, err := repo.CreateDevice(ctx, "device-1")
	is.NoErr(err)
	_, err = repo.CreateDevice(ctx, "device-2")
	is.NoErr(err)

	// Get all devices
	devices, err := repo.GetDevices(ctx)
	is.NoErr(err)
	is.Equal(len(devices), 2) // Should have 2 devices
}

func TestRepository_CreateDevice_DuplicateName(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create first device
	_, err := repo.CreateDevice(ctx, "duplicate-name")
	is.NoErr(err)

	// Try to create device with same name
	_, err = repo.CreateDevice(ctx, "duplicate-name")
	is.True(err != nil) // Should error (UNIQUE constraint)
}

func TestRepository_DatabaseIsolation(t *testing.T) {

	// Test 1: Create 1 device
	t.Run("test1", func(t *testing.T) {
		is := is.New(t)

		repo := setupTestDB(t)
		ctx := context.Background()

		repo.CreateDevice(ctx, "device-1")

		devices, err := repo.GetDevices(ctx)
		is.NoErr(err)
		is.Equal(len(devices), 1) // Should have 1 device
	})

	// Test 2: Should have 0 devices (fresh DB)
	t.Run("test2", func(t *testing.T) {
		is := is.New(t)

		repo := setupTestDB(t)
		ctx := context.Background()

		devices, err := repo.GetDevices(ctx)
		is.NoErr(err)
		is.Equal(len(devices), 0) // Should be empty (isolated from test1)
	})
}
