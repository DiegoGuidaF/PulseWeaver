package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// StartAndWait starts the HTTP server in the background.
// It blocks until a server error occurs or the provided context is cancelled,
// at which point it performs a graceful shutdown with the configured timeout.
func StartAndWait(ctx context.Context, handler http.Handler, cfg ServerConfig, logger *slog.Logger) error {
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

	// Graceful Shutdown: wait for server error or context cancellation
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server startup error: %w", err)
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
