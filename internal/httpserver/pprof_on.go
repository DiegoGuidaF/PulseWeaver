//go:build pprof

package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"time"
)

// pprofAddr is the loopback address the debug profiling server binds to. It is
// deliberately hardcoded and not configurable: the listener is compiled in only
// under the `pprof` build tag, and binding to loopback keeps the surface
// unreachable from off-host and from the reverse proxy in front of the API.
const pprofAddr = "127.0.0.1:6060"

// StartPprofServer runs a loopback-only HTTP server exposing net/http/pprof. It
// is compiled in only under the `pprof` build tag (see pprof_off.go for the
// default no-op). Handlers are registered explicitly on a private ServeMux
// rather than via net/http/pprof's blank-import side effect on DefaultServeMux,
// so nothing leaks onto the main application server.
//
// The server runs with a zero WriteTimeout so a long CPU or trace capture
// (profile?seconds=30) is not truncated — unlike the main server, which keeps
// its prod WriteTimeout. It blocks until ctx is cancelled, then shuts down
// gracefully, mirroring StartAndWait.
func StartPprofServer(ctx context.Context, logger *slog.Logger) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	srv := &http.Server{
		Addr:              pprofAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		// WriteTimeout is intentionally left zero so 30s+ profiles aren't cut off.
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Warn("pprof debug server listening — profiling build, do not deploy", slog.String("addr", pprofAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("pprof server error: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("pprof graceful shutdown failed: %w", err)
	}
	return nil
}
