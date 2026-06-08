package database

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	sqlitedriver "modernc.org/sqlite"
)

// ErrContended is returned when a write could not proceed because SQLite's
// single writer was held and busy_timeout expired. It is an availability
// signal, not a domain error: the HTTP boundary maps it to 503 Service
// Unavailable with a Retry-After header, never a 500.
var ErrContended = errors.New("database write contention")

type contentionFlagKey struct{}

// WithContentionFlag returns a context carrying a contention flag plus the flag
// itself. withinTx sets the flag when a write transaction fails with
// ErrContended; a caller (the HTTP layer) reads it after the work completes to
// translate the response — out of band, since the contention error is otherwise
// swallowed by handlers' own error mapping.
func WithContentionFlag(ctx context.Context) (context.Context, *atomic.Bool) {
	flag := &atomic.Bool{}
	return context.WithValue(ctx, contentionFlagKey{}, flag), flag
}

// markContended sets the contention flag if one is present on the context.
func markContended(ctx context.Context) {
	if flag, ok := ctx.Value(contentionFlagKey{}).(*atomic.Bool); ok {
		flag.Store(true)
	}
}

// SQLite extended result codes for lock contention. With _txlock=immediate the
// un-waitable BUSY_SNAPSHOT (517) should no longer occur, but it is matched here
// for completeness alongside the plain BUSY (5) and the transient BUSY_RECOVERY
// (261) raised during WAL recovery.
const (
	sqliteBusy         = 5   // SQLITE_BUSY
	sqliteBusyRecovery = 261 // SQLITE_BUSY_RECOVERY
	sqliteBusySnapshot = 517 // SQLITE_BUSY_SNAPSHOT
)

// mapBusyErr translates a SQLite lock-contention error into ErrContended,
// preserving the original message for logging. Any other error passes through
// unchanged.
func mapBusyErr(err error) error {
	if err == nil {
		return nil
	}
	if se, ok := errors.AsType[*sqlitedriver.Error](err); ok {
		switch se.Code() {
		case sqliteBusy, sqliteBusyRecovery, sqliteBusySnapshot:
			return fmt.Errorf("%w: %w", ErrContended, err)
		}
	}
	return err
}
