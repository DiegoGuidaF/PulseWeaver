//go:build test

package testutils

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/app"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
)

// SetupIntegrationServer creates a complete integration test server with database,
// services, and handlers configured.
func SetupIntegrationServer(t *testing.T) *app.App {
	t.Helper()

	conf := &config.Conf{
		Server: config.ConfServer{
			Port:          2000,
			AdminPassword: "AdminPass123!",
			TrustedProxy:  "127.0.0.1",
		},
		DB: config.ConfDB{
			Dsn:   fmt.Sprintf("file:%s?mode=memory&_loc=auto", t.Name()),
			Debug: false,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	application, err := app.NewWithConfigAndLogger(context.Background(), conf, logger)
	if err != nil {
		t.Fatalf("setup app: %v", err)
	}

	t.Cleanup(func() {
		if err := application.Close(); err != nil {
			t.Logf("error closing app: %v", err)
		}
	})

	return application
}
