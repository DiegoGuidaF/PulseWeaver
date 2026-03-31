//go:build test

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Validation(t *testing.T) {
	tmpDir := t.TempDir()
	// Set required env vars as a baseline for all subtests.
	t.Setenv("ADMIN_PASSWORD", "TestAdminPassword1!")
	t.Setenv("POLICY_ENGINE_API_SECRET", "averylongandsecuresecret")
	t.Setenv("DB_DIR", tmpDir)
	t.Setenv("GEOIP_DATA_DIR", tmpDir)

	t.Run("valid config loads with expected defaults", func(t *testing.T) {
		conf, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if conf.Server.Port != 8080 {
			t.Errorf("Port = %d, want 8080", conf.Server.Port)
		}
		if conf.LogLevel != "info" {
			t.Errorf("LogLevel = %q, want info", conf.LogLevel)
		}
		if conf.Rules.CheckInterval.String() != "1m0s" {
			t.Errorf("CheckInterval = %s, want 1m0s", conf.Rules.CheckInterval)
		}
		if conf.Server.TZ != "UTC" {
			t.Errorf("TZ = %q, want UTC", conf.Server.TZ)
		}
	})

	t.Run("POLICY_ENGINE_API_SECRET shorter than 16 chars fails", func(t *testing.T) {
		t.Setenv("POLICY_ENGINE_API_SECRET", "tooshort")
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for short API secret, got nil")
		}
	})

	t.Run("SERVER_PORT below range fails", func(t *testing.T) {
		t.Setenv("SERVER_PORT", "0")
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for port 0, got nil")
		}
	})

	t.Run("SERVER_PORT above range fails", func(t *testing.T) {
		t.Setenv("SERVER_PORT", "99999")
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for port 99999, got nil")
		}
	})

	t.Run("zero RULE_CHECK_INTERVAL fails", func(t *testing.T) {
		t.Setenv("RULE_CHECK_INTERVAL", "0s")
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for zero check interval, got nil")
		}
	})

	t.Run("negative RULE_CHECK_INTERVAL fails", func(t *testing.T) {
		t.Setenv("RULE_CHECK_INTERVAL", "-1m")
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for negative check interval, got nil")
		}
	})
	t.Run("non existent DB_DIR failsfails", func(t *testing.T) {
		t.Setenv("DB_DIR", "a non existant db dir")

		_, err := Load()
		if err == nil {
			t.Fatal("expected error for non existent DB_DIR, got nil")
		}
	})

	t.Run("unwritable DB_DIR fails", func(t *testing.T) {
		// Create a read-only directory
		readonlyDir := filepath.Join(t.TempDir(), "readonly")
		if err := os.Mkdir(readonlyDir, 0555); err != nil { // Read-only
			t.Fatalf("setup readonly dir: %v", err)
		}

		t.Setenv("DB_DIR", readonlyDir)
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for unwritable DB_DIR, got nil")
		}
	})

	t.Run("non existent GEOIP_DATA_DIR fails", func(t *testing.T) {
		t.Setenv("GEOIP_DATA_DIR", "a non existant dir")

		_, err := Load()
		if err == nil {
			t.Fatal("expected error for non existent GEOIP_DATA_DIR, got nil")
		}
	})

	t.Run("unwritable DB_DIR fails", func(t *testing.T) {
		// Create a read-only directory
		readonlyDir := filepath.Join(t.TempDir(), "readonly")
		if err := os.Mkdir(readonlyDir, 0555); err != nil { // Read-only
			t.Fatalf("setup readonly dir: %v", err)
		}

		t.Setenv("GEOIP_DATA_DIR", readonlyDir)
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for unwritable GEOIP_DATA_DIR, got nil")
		}
	})
}
