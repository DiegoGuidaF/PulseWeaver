package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/auth"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpserver"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
)

// App holds all initialized application components.
type App struct {
	Config        *config.Conf
	Logger        *slog.Logger
	Database      *database.SQLite
	HTTPServer    http.Handler
	DeviceService *device.Service
	AuthService   *auth.Service
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
// If logger is nil, a default logger will be created based on the configured environment.
func NewWithConfigAndLogger(ctx context.Context, conf *config.Conf, logger *slog.Logger) (*App, error) {
	if logger == nil {
		logger = logging.New(conf.Environment)
	}

	// Set logger for dependencies
	slog.SetDefault(logger)

	// Log startup configuration
	logger.Info("initializing app",
		slog.Int("port", conf.Server.Port),
		slog.String("environment", conf.Environment),
		slog.String("db_file", conf.DB.File),
	)

	// Database Connection
	db, err := database.NewSQLite(conf.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migration error: %w", err)
	}
	logger.Info("database initialized and connected successfully")

	// Dependency Injection
	deviceRepo := device.NewRepository(db.DB())
	deviceService := device.NewService(deviceRepo)
	openApiHandler := device.NewOpenApiHandler(deviceService, logger)

	authRepo := auth.NewRepository(db.DB())
	authService := auth.NewService(authRepo, logger)
	authHandler := auth.NewHandler(authService, logger)

	err = authService.BootstrapAdmin(ctx, conf.Server)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to bootstrap admin: %w", err)
	}

	handler, err := httpserver.NewServerFromConfig(openApiHandler, authHandler, logger, conf.Server)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create http server: %w", err)
	}

	return &App{
		Config:        conf,
		Logger:        logger,
		Database:      db,
		HTTPServer:    handler,
		DeviceService: deviceService,
		AuthService:   authService,
	}, nil
}

// Close cleans up application resources.
func (a *App) Close() error {
	if a.Database != nil {
		return a.Database.Close()
	}
	return nil
}
