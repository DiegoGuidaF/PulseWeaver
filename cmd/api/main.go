package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/app"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpserver"
)

func run(ctx context.Context) (*slog.Logger, error) {
	// Initialize application
	application, err := app.New(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := application.Close(); err != nil {
			// Log error if logger is available, otherwise ignore
			if logger := application.Logger; logger != nil {
				logger.Error("failed to close application", slog.Any("error", err))
			}
		}
	}()

	serverConfig := httpserver.DefaultServerConfigFromConf(application.Config.Server.Port)
	if err := httpserver.StartAndWait(ctx, application.HTTPServer, serverConfig, application.Logger); err != nil {
		return application.Logger, err
	}

	return application.Logger, nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger, err := run(ctx)
	if err != nil {
		// If logger hasn't been instantiated do it now
		if logger == nil {
			logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
		}
		logger.Error("application exited with error", slog.Any("error", err))
		os.Exit(1)
	}
}
