package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog" // Import slog
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpserver"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
)

func run(ctx context.Context) (*slog.Logger, error) {
	conf, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config load error: %w", err)
	}

	// 1. Initialize Logger
	logger := logging.New(conf.Environment)

	// Set as default logger for dependencies
	slog.SetDefault(logger)

	// Log startup configuration
	logger.Info("starting server",
		slog.Int("port", conf.Server.Port),
		slog.String("environment", conf.Environment),
		slog.String("db_file", conf.DB.File),
	)

	// 2. Database Connection
	db, err := database.NewSQLite(&conf.DB)
	if err != nil {
		return logger, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		return logger, fmt.Errorf("migration error: %w", err)
	}
	logger.Info("database initialized and connected successfully")

	// 3. Dependency Injection
	deviceRepo := device.NewRepository(db)
	deviceService := device.NewService(deviceRepo)
	deviceHandler := device.NewHandler(deviceService, logger)

	handler := httpserver.NewServer(deviceHandler, logger)

	// 4. Setup HTTP Server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", conf.Server.Port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 5. Start Server in Goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// 6. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return logger, fmt.Errorf("server startup error: %w", err)
	case sig := <-quit:
		logger.Info("shutting down server", slog.String("signal", sig.String()))

		// Give outstanding requests 5 seconds to complete
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			return logger, fmt.Errorf("graceful shutdown failed: %w", err)
		}
	}

	logger.Info("server stopped")
	return logger, nil
}

func main() {
	ctx := context.Background()
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
