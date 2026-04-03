package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/config"
	"github.com/DiegoGuidaF/PulseWeaver/internal/dashboard"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpserver"
	"github.com/DiegoGuidaF/PulseWeaver/internal/lease"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/maxaddr"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries"
	"github.com/DiegoGuidaF/PulseWeaver/internal/registration"
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
	RegistrationService *registration.Service
	addressLeaseService *lease.Service
	maxAddrService      *maxaddr.Service
	schedulerService    *scheduler.Service
	accessLogSink       *accesslog.Sink
	geoipLookup         *geoip.Lookup
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

	// Device provisioner
	registrationRepo := registration.NewRepository(db.DB())
	registrationService := registration.NewService(registrationRepo, logger)
	registrationHandler := registration.NewHTTPHandler(registrationService, logger)

	// GeoIP enrichment
	geoipLookup, err := geoip.New(ctx, conf.GeoIP, logger)
	if err != nil {
		return nil, fmt.Errorf("geoip init: %w", err)
	}

	// Policy forward-auth sidecar
	policyService, err := policy.NewService(deviceService, geoipLookup, conf.Policy.APISecret, logger, conf.Server.TrustedProxy)
	if err != nil {
		return nil, fmt.Errorf("policy service init: %w", err)
	}
	policyHandler := policy.NewHTTPHandler(policyService, logger)

	// Rule evaluation
	ruleRepo := rule.NewRepository(db.DB())
	ruleService := rule.NewService(ruleRepo, logger)
	ruleHandler := rule.NewHTTPHandler(ruleService, logger)

	// Access log
	accessLogRepo := accesslog.NewRepository(db.DB())
	accessLogSink := accesslog.NewSink(accessLogRepo, logger)
	accessLogHandler := accesslog.NewHTTPHandler(accessLogRepo, logger)

	// Queries - Manage complex crossdomain queries tailored for the frontend
	queriesRepo := queries.NewRepository(db.DB())
	queriesHandler := queries.NewHTTPHandler(queriesRepo, logger)

	// Address Lease manager
	addressLeaseRepo := lease.NewRepository(db.DB())
	addressLeaseService := lease.NewService(addressLeaseRepo, ruleService, logger)

	// Max active addresses enforcer
	maxAddrService := maxaddr.NewService(ruleService, deviceService, deviceService, logger)

	// Register device address observers
	deviceService.AddAddressObserver(addressLeaseService)
	deviceService.AddAddressObserver(policyService)
	deviceService.AddAddressObserver(maxAddrService)

	// Register rule change observers
	ruleService.AddRuleObserver(addressLeaseService)
	ruleService.AddRuleObserver(maxAddrService)

	// Register policy decision observers
	policyService.AddDecisionObserver(accessLogSink)

	// Dashboard — traffic aggregation
	dashboardRepo := dashboard.NewRepository(db.DB())
	dashboardHandler := dashboard.NewHTTPHandler(dashboardRepo, logger)

	// Runs scheduled jobs - Address leasing, traffic aggregates for the dashboard...
	schedulerService, err := scheduler.NewService(addressLeaseService, deviceService, dashboardRepo, logger)
	if err != nil {
		return nil, fmt.Errorf("scheduler service init: %w", err)
	}

	// Fire the rules on start to ensure we disable no longer valid addresses before letting them through
	err = schedulerService.ExecuteScheduledRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run scheduled rules on init: %w", err)
	}

	// Ensure there's at least 1 admin user
	err = authService.BootstrapAdmin(ctx, conf.Server)
	if err != nil {
		return nil, fmt.Errorf("failed to bootstrap admin: %w", err)
	}

	// Initialize the authorization hotpath by loading the enabled addresses in-memory
	if err := policyService.Initialize(ctx); err != nil {
		logger.Warn("failed to initialize policy IP cache on startup", slog.Any("error", err))
	}

	handler := httpserver.NewServer(deviceHandler, authHandler, ruleHandler, queriesHandler, policyHandler, accessLogHandler, dashboardHandler, registrationHandler, logger, conf.Server.TrustedProxy)

	return &App{
		Config:              conf,
		Logger:              logger,
		Database:            db,
		HTTPServer:          handler,
		DeviceService:       deviceService,
		AuthService:         authService,
		PolicyService:       policyService,
		RuleService:         ruleService,
		RegistrationService: registrationService,
		addressLeaseService: addressLeaseService,
		maxAddrService:      maxAddrService,
		schedulerService:    schedulerService,
		accessLogSink:       accessLogSink,
		geoipLookup:         geoipLookup,
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
		return ignoreContextCanceled(a.maxAddrService.RunListener(gCtx))
	})

	g.Go(func() error {
		return ignoreContextCanceled(a.schedulerService.RunSchedule(gCtx, a.Config.Rules.CheckInterval))
	})

	g.Go(func() error {
		return ignoreContextCanceled(a.accessLogSink.Run(gCtx))
	})

	g.Go(func() error {
		return ignoreContextCanceled(a.geoipLookup.RunUpdater(gCtx, a.Logger))
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
