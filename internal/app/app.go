package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/auth"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/caddy"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpserver"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/lease"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/rule"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/scheduler"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/whitelist"
)

// App holds all initialized application components.
type App struct {
	Config           *config.Conf
	Logger           *slog.Logger
	Database         *database.SQLite
	HTTPServer       http.Handler
	DeviceService    *device.Service
	AuthService      *auth.Service
	WhitelistService *whitelist.Service
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

	// Authentication
	authRepo := auth.NewRepository(db.DB())
	authService := auth.NewService(authRepo)
	authHandler := auth.NewHandler(authService)

	// Device & addresses management
	deviceRepo := device.NewRepository(db.DB())
	deviceService := device.NewService(deviceRepo)
	deviceHandler := device.NewHandler(deviceService)

	// Rule evaluation
	ruleRepo := rule.NewRepository(db.DB())
	ruleService := rule.NewService(ruleRepo)
	ruleHandler := rule.NewHandler(ruleService)

	// Whitelist generation
	var whitelistChangeNotifier whitelist.ChangeNotifier
	if conf.Caddy.Endpoint != "" {
		caddyReloadClient := caddy.NewReloaderClient(conf.Caddy.Endpoint, conf.Caddy.AuthToken)
		whitelistChangeNotifier = caddyReloadClient
		go func() {
			if err := caddyReloadClient.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				logger.Error("whitelist change notifier exited with error", slog.Any("error", err))
			}
		}()
	} else {
		whitelistChangeNotifier = whitelist.NoOpNotifier{}
		logger.Info("whitelist change webhook URL not configured; change notifications disabled")
	}

	whitelistService := whitelist.NewService(deviceService, conf.Whitelist, whitelistChangeNotifier)

	// Address Lease manager
	addressLeaseRepo := lease.NewRepository(db.DB())
	addressLeaseService := lease.NewService(addressLeaseRepo, ruleService)

	// Register device address observers
	deviceService.AddAddressObserver(whitelistService)
	deviceService.AddAddressObserver(addressLeaseService)

	schedulerService, err := scheduler.NewService(addressLeaseService, deviceService)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("scheduler service init: %w", err)
	}

	err = authService.BootstrapAdmin(ctx, conf.Server)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to bootstrap admin: %w", err)
	}

	// Initial whitelist generation
	if err := whitelistService.Regenerate(ctx); err != nil {
		logger.Warn("failed to generate whitelist on startup", slog.Any("error", err))
	}

	handler, err := httpserver.NewServerFromConfig(deviceHandler, authHandler, ruleHandler, logger, conf.Server)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create http server: %w", err)
	}

	// Start whitelist generation listener
	go func() {
		if err := whitelistService.RunListener(ctx); err != nil {
			logger.Error("whitelist service exited with error", slog.Any("error", err))
		}
	}()

	// Start address lease listener
	go func() {
		if err := addressLeaseService.RunListener(ctx); err != nil {
			logger.Error("address lease service exited with error", slog.Any("error", err))
		}
	}()

	// Start rule scheduler for time-based rules (e.g., IP auto-expiry)
	if conf.Rules.CheckInterval > 0 {
		go func() {
			if err := schedulerService.RunSchedule(ctx, conf.Rules.CheckInterval); err != nil && !errors.Is(err, context.Canceled) {
				logger.Error("rule scheduler exited with error", slog.Any("error", err))
			}
		}()
	} else {
		logger.Warn("rule scheduler disabled due to non-positive check interval", slog.Duration("interval", conf.Rules.CheckInterval))
	}

	return &App{
		Config:           conf,
		Logger:           logger,
		Database:         db,
		HTTPServer:       handler,
		DeviceService:    deviceService,
		AuthService:      authService,
		WhitelistService: whitelistService,
	}, nil
}

// Close cleans up application resources.
func (a *App) Close() error {
	if a.Database != nil {
		return a.Database.Close()
	}
	return nil
}
