package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/DiegoGuidaF/PulseWeaver/internal/audit"
	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/config"
	"github.com/DiegoGuidaF/PulseWeaver/internal/dashboard"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpserver"
	"github.com/DiegoGuidaF/PulseWeaver/internal/lease"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
	"github.com/DiegoGuidaF/PulseWeaver/internal/scheduler"
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
	PolicyService       *policy.Service
	RuleService         *rule.Service
	addressLeaseService *lease.Service
	schedulerService    *scheduler.Service
	auditSink           *audit.Sink
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
		slog.String("db_dir", conf.DB.DataDir),
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

	// Policy forward-auth sidecar
	policyService, err := policy.NewService(deviceService, conf.Policy.APISecret, logger, conf.Server.TrustedProxy)
	if err != nil {
		return nil, fmt.Errorf("policy service init: %w", err)
	}
	policyHandler := policy.NewHTTPHandler(policyService, logger)

	// Rule evaluation
	ruleRepo := rule.NewRepository(db.DB())
	ruleService := rule.NewService(ruleRepo, logger)
	ruleHandler := rule.NewHTTPHandler(ruleService, logger)

	// Audit log — write side + simple reads (deny reasons)
	auditRepo := audit.NewRepository(db.DB())
	auditSink := audit.NewSink(auditRepo, logger)
	auditHandler := audit.NewHTTPHandler(auditRepo, logger)

	queriesRepo := queries.NewRepository(db.DB())
	queriesHandler := queries.NewHTTPHandler(queriesRepo, logger)

	// Address Lease manager
	addressLeaseRepo := lease.NewRepository(db.DB())
	addressLeaseService := lease.NewService(addressLeaseRepo, ruleService, logger)

	// Register device address observers
	deviceService.AddAddressObserver(addressLeaseService)
	ruleService.AddRuleObserver(addressLeaseService)
	deviceService.AddAddressObserver(policyService)

	// Register policy decision observers
	policyService.AddDecisionObserver(auditSink)

	// Dashboard — traffic aggregation
	dashboardRepo := dashboard.NewRepository(db.DB())
	dashboardHandler := dashboard.NewHTTPHandler(dashboardRepo, logger)

	schedulerService, err := scheduler.NewService(addressLeaseService, deviceService, dashboardRepo, logger)
	if err != nil {
		return nil, fmt.Errorf("scheduler service init: %w", err)
	}

	err = schedulerService.ExecuteScheduledRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run scheduled rules on init: %w", err)
	}

	err = authService.BootstrapAdmin(ctx, conf.Server)
	if err != nil {
		return nil, fmt.Errorf("failed to bootstrap admin: %w", err)
	}

	if err := policyService.Initialize(ctx); err != nil {
		logger.Warn("failed to initialize policy IP cache on startup", slog.Any("error", err))
	}

	handler := httpserver.NewServer(deviceHandler, authHandler, ruleHandler, queriesHandler, policyHandler, auditHandler, dashboardHandler, logger, conf.Server.TrustedProxy)

	return &App{
		Config:              conf,
		Logger:              logger,
		Database:            db,
		HTTPServer:          handler,
		DeviceService:       deviceService,
		AuthService:         authService,
		PolicyService:       policyService,
		RuleService:         ruleService,
		addressLeaseService: addressLeaseService,
		schedulerService:    schedulerService,
		auditSink:           auditSink,
	}, nil
}

// Run starts all application background services and the HTTP server.
// It blocks until shutdown or the first non-cancelled error.
func (a *App) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return ignoreContextCanceled(a.PolicyService.RunListener(gCtx))
	})

	g.Go(func() error {
		return ignoreContextCanceled(a.addressLeaseService.RunListener(gCtx))
	})

	g.Go(func() error {
		return ignoreContextCanceled(a.schedulerService.RunSchedule(gCtx, a.Config.Rules.CheckInterval))
	})

	g.Go(func() error {
		return ignoreContextCanceled(a.auditSink.Run(gCtx))
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
