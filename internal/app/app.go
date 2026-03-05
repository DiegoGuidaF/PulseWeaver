package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

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
	"golang.org/x/sync/errgroup"
)

// App holds all initialized application components.
type App struct {
	Config              *config.Conf
	Logger              *slog.Logger
	Database            *database.SQLite
	HTTPServer          http.Handler
	DeviceService       *device.Service
	AuthService         *auth.Service
	AuthzService        *authz.Service
	addressLeaseService *lease.Service
	schedulerService    *scheduler.Service
}

// New initializes the application with configuration loaded from environment variables.
func New(ctx context.Context) (*App, error) {
	conf, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config load error: %w", err)
	}

	logger := logging.New(logging.Options{
		Level:  logging.ParseLevel(conf.LogLevel),
		Format: conf.LogFormat,
		Color:  conf.LogColor,
	})

	return NewWithConfigAndLogger(ctx, conf, logger)
}

// NewWithConfigAndLogger initializes the application with the provided configuration and logger.
func NewWithConfigAndLogger(ctx context.Context, conf *config.Conf, logger *slog.Logger) (_ *App, err error) {
	// Set logger for dependencies
	slog.SetDefault(logger)

	if !conf.Server.TrustedProxy.IsValid() {
		logger.Warn("TRUSTED_PROXY is not configured — if this service is running behind a reverse proxy " +
			"(Caddy, Nginx, Traefik, etc.), all client IPs will be detected as the proxy's IP address. " +
			"Set TRUSTED_PROXY to your proxy's IP. " +
			"Ignore this warning only if clients connect directly with no proxy in front.")
	}

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
	defer func() {
		if err != nil {
			_ = db.Close()
		}
	}()

	if err = db.Migrate(); err != nil {
		return nil, fmt.Errorf("migration error: %w", err)
	}
	logger.Info("database initialized and connected successfully")

	// Authentication
	authRepo := auth.NewRepository(db.DB())
	authService := auth.NewService(authRepo, logger)
	authHandler := auth.NewHandler(authService, logger)

	// Device & addresses management
	deviceRepo := device.NewRepository(db.DB())
	deviceService := device.NewService(deviceRepo, logger, conf.Server.TrustedProxy)
	deviceHandler := device.NewHTTPHandler(deviceService, logger)

	// Authz forward-auth sidecar
	authzService, err := authz.NewService(deviceService, conf.Authz.APISecret, logger, conf.Server.TrustedProxy)
	if err != nil {
		return nil, fmt.Errorf("authz service init: %w", err)
	}
	authzHandler := authz.NewHTTPHandler(authzService, logger)

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
		return nil, fmt.Errorf("scheduler service init: %w", err)
	}

	err = authService.BootstrapAdmin(ctx, conf.Server)
	if err != nil {
		return nil, fmt.Errorf("failed to bootstrap admin: %w", err)
	}

	if err := authzService.Initialize(ctx); err != nil {
		logger.Warn("failed to initialize authz IP cache on startup", slog.Any("error", err))
	}

	handler := httpserver.NewServer(deviceHandler, authHandler, ruleHandler, authzHandler, logger, conf.Server.TrustedProxy)

	return &App{
		Config:              conf,
		Logger:              logger,
		Database:            db,
		HTTPServer:          handler,
		DeviceService:       deviceService,
		AuthService:         authService,
		AuthzService:        authzService,
		addressLeaseService: addressLeaseService,
		schedulerService:    schedulerService,
	}, nil
}

// Run starts all application background services and the HTTP server.
// It blocks until shutdown or the first non-cancelled error.
func (a *App) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return ignoreContextCanceled(a.AuthzService.RunListener(gCtx))
	})

	g.Go(func() error {
		return ignoreContextCanceled(a.addressLeaseService.RunListener(gCtx))
	})

	g.Go(func() error {
		return ignoreContextCanceled(a.schedulerService.RunSchedule(gCtx, a.Config.Rules.CheckInterval))
	})

	serverConfig := httpserver.DefaultServerConfigFromConf(a.Config.Server.Port)
	g.Go(func() error {
		return ignoreContextCanceled(httpserver.StartAndWait(gCtx, a.HTTPServer, serverConfig, a.Logger))
	})

	return ignoreContextCanceled(g.Wait())
}

func ignoreContextCanceled(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

// Close cleans up application resources.
func (a *App) Close() error {
	if a.Database != nil {
		return a.Database.Close()
	}
	return nil
}
