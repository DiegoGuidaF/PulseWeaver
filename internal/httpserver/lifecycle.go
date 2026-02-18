package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// StartAndWait starts the HTTP server and waits for it to shut down gracefully.
// It handles signal listening (SIGINT, SIGTERM) and implements graceful shutdown
// with the configured timeout.
func StartAndWait(ctx context.Context, handler http.Handler, cfg *ServerConfig, logger *slog.Logger) error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}

	// Start Server in Goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server startup error: %w", err)
	case sig := <-quit:
		logger.Info("shutdown signal received, shutting down server", slog.String("signal", sig.String()))
	case <-ctx.Done():
		logger.Info("shutting down server (context cancelled)")
	}

	// Give outstanding requests time to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	logger.Info("server stopped")
	return nil
}
