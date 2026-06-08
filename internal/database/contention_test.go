//go:build test

package database

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/config"
)

// TestWithinTx_Contention verifies that when the single SQLite writer is held,
// a concurrent write transaction surfaces as ErrContended (not a raw SQLITE_BUSY)
// and that the contention flag carried on the context is set. busy_timeout is
// dialled down to 100ms so the blocked writer fails fast instead of waiting the
// production 5s.
func TestWithinTx_Contention(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "contention.db")
	dsn := fmt.Sprintf(
		"file:%s?_time_format=sqlite&_texttotime=1&_timezone=UTC"+
			"&_txlock=immediate&_pragma=journal_mode(WAL)&_pragma=busy_timeout(100)",
		dbPath,
	)

	sq, err := NewSQLite(config.ConfDB{Dsn: dsn})
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	t.Cleanup(func() { _ = sq.Close() })

	db := sq.DB()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Goroutine A grabs the writer (BEGIN IMMEDIATE) and holds it open until
	// released, guaranteeing the concurrent writer contends.
	holding := make(chan struct{})
	release := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		_ = db.WithinTx(ctx, func(ctx context.Context) error {
			if _, err := db.ExecContext(ctx, `INSERT INTO t (v) VALUES ('a')`); err != nil {
				return err
			}
			close(holding)
			<-release
			return nil
		})
	})

	<-holding // A now holds the write lock

	ctxB, contended := WithContentionFlag(ctx)
	errB := db.WithinTx(ctxB, func(ctx context.Context) error {
		_, err := db.ExecContext(ctx, `INSERT INTO t (v) VALUES ('b')`)
		return err
	})

	close(release)
	wg.Wait()

	if !errors.Is(errB, ErrContended) {
		t.Fatalf("expected ErrContended, got %v", errB)
	}
	if !contended.Load() {
		t.Fatal("expected contention flag to be set on the context")
	}
}
