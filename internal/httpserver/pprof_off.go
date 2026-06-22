//go:build !pprof

package httpserver

import (
	"context"
	"log/slog"
)

// StartPprofServer is a no-op in the default build: the pprof debug server is
// compiled in only under the `pprof` build tag (see pprof_on.go). The handlers
// are physically absent from any binary built without the tag, so there is
// nothing to misconfigure or enable in a real deployment.
func StartPprofServer(_ context.Context, _ *slog.Logger) error {
	return nil
}
