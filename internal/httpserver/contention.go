package httpserver

import (
	"context"
	"net/http"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
)

// contentionMiddleware translates SQLite write-lock contention into a 503 in one
// place, so no domain handler needs to know about it.
//
// It attaches a contention flag to the request context before the handler runs;
// database.withinTx sets that flag when a write transaction fails with
// ErrContended. After the handler returns, if the flag is set we discard the
// handler's response (typically a generic 500) and hand database.ErrContended to
// the strict wrapper, which routes it through createResponseErrorHandler →
// 503 + Retry-After. Returning an error here means the wrapper never calls the
// handler's response Visit method, so nothing has been written to w yet.
func contentionMiddleware(f httpapi.StrictHandlerFunc, _ string) httpapi.StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		ctx, contended := database.WithContentionFlag(ctx)

		response, err := f(ctx, w, r, request)
		if err == nil && contended.Load() {
			return nil, database.ErrContended
		}
		return response, err
	}
}
