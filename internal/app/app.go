package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/DiegoGuidaF/WallyDex/internal/auth"
	"github.com/DiegoGuidaF/WallyDex/internal/authz"
	"github.com/DiegoGuidaF/WallyDex/internal/config"
	"github.com/DiegoGuidaF/WallyDex/internal/database"
	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/DiegoGuidaF/WallyDex/internal/httpserver"
	"github.com/DiegoGuidaF/WallyDex/internal/lease"
	"github.com/DiegoGuidaF/WallyDex/internal/logging"
	"github.com/DiegoGuidaF/WallyDex/internal/rule"
	"github.com/DiegoGuidaF/WallyDex/internal/scheduler"
)

// App holds all initialized application components.
type App struct {
	Config        *config.Conf
	Logger        *slog.Logger
	Database      *database.SQLite
	HTTPServer    http.Handler
	DeviceService *device.Service
	AuthService   *auth.Service
	AuthzService  *authz.Service
	wg            sync.WaitGroup
}

// New initializes the application with configuration loaded from environment variables.
func New(ctx context.Context) (*App, error) {
	conf, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config load error: %w", err)
	}

	return NewWithConfig(ctx, conf)
}

// NewWithConfig initializes the application with the provided configuration.
// This is useful for testing where custom configuration (e.g., in-memory database) is needed.
func NewWithConfig(ctx context.Context, conf *config.Conf) (*App, error) {
	return NewWithConfigAndLogger(ctx, conf, nil)
}

// NewWithConfigAndLogger initializes the application with the provided configuration and logger.
// If logger is nil, a default logger is created from LOG_LEVEL and LOG_FORMAT.
func NewWithConfigAndLogger(ctx context.Context, conf *config.Conf, logger *slog.Logger) (*App, error) {
	if logger == nil {
		logger = logging.New(logging.Options{
			Level:  logging.ParseLevel(conf.LogLevel),
			Format: conf.LogFormat,
			Color:  conf.LogColor,
		})
	}

	// Set logger for dependencies
	slog.SetDefault(logger)

	// Log startup configuration
	logger.Info("initializing app",
		slog.Int("port", conf.Server.Port),
		slog.String("db_file", conf.DB.File),
	)

	// Database Connection
	db, err := database.NewSQLite(conf.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migration error: %w", err)
	}
	logger.Info("database initialized and connected successfully")

	a := &App{
		Config:   conf,
		Logger:   logger,
		Database: db,
	}

	// Authentication
	authRepo := auth.NewRepository(db.DB())
	authService := auth.NewService(authRepo, logger)
	authHandler := auth.NewHandler(authService, logger)

	// Device & addresses management
	deviceRepo := device.NewRepository(db.DB())
	deviceService := device.NewService(deviceRepo, logger)
	deviceHandler := device.NewHandler(deviceService, logger)

	// Authz forward-auth sidecar
	authzService := authz.NewService(deviceService, conf.Authz.APISecret, logger)
	authzHandler := authz.NewHandler(authzService, logger)

	// Rule evaluation
	ruleRepo := rule.NewRepository(db.DB())
	ruleService := rule.NewService(ruleRepo, logger)
	ruleHandler := rule.NewHandler(ruleService, logger)

	// Address Lease manager
	addressLeaseRepo := lease.NewRepository(db.DB())
	addressLeaseService := lease.NewService(addressLeaseRepo, ruleService, logger)

	// Register device address observers
	deviceService.AddAddressObserver(addressLeaseService)
	deviceService.AddAddressObserver(authzService)

	schedulerService, err := scheduler.NewService(addressLeaseService, deviceService, logger)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("scheduler service init: %w", err)
	}

	err = authService.BootstrapAdmin(ctx, conf.Server)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to bootstrap admin: %w", err)
	}

	if err := authzService.Initialize(ctx); err != nil {
		logger.Warn("failed to initialize authz IP cache on startup", slog.Any("error", err))
	}

	handler, err := httpserver.NewServerFromConfig(deviceHandler, authHandler, ruleHandler, authzHandler, logger, conf.Server)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create http server: %w", err)
	}

	// Start authz address change listener to rebuild IP registry cache
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := authzService.RunListener(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("authz service exited with error", slog.Any("error", err))
		}
	}()

	// Start address lease listener
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := addressLeaseService.RunListener(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("address lease service exited with error", slog.Any("error", err))
		}
	}()

	// Start rule scheduler for time-based rules (e.g., IP auto-expiry)
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := schedulerService.RunSchedule(ctx, conf.Rules.CheckInterval); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("scheduler exited with error", slog.Any("error", err))
		}
	}()

	a.HTTPServer = handler
	a.DeviceService = deviceService
	a.AuthService = authService
	a.AuthzService = authzService

	return a, nil
}

// Close waits for all background goroutines to finish, then cleans up application resources.
func (a *App) Close() error {
	a.wg.Wait()
	if a.Database != nil {
		return a.Database.Close()
	}
	return nil
}
