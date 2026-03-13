package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/jmoiron/sqlx"

	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
)

// Repository owns the write side of the audit log.
// The read side lives in the internal/queries package.
type Repository struct {
	db     dBInterface
	rootDB *sqlx.DB
}

type dBInterface interface {
	sqlx.ExtContext
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		rootDB: db,
		db:     db,
	}
}

func (r *Repository) BatchInsert(ctx context.Context, events []policy.DecisionEvent) error {
	if len(events) == 0 {
		return nil
	}

	return r.runInTx(ctx, func(tx *Repository) error {

		const query = `
		INSERT INTO request_audit_log (
			client_ip,
			device_id,
			address_id,
			outcome,
			deny_reason,
			created_at,
			xff_chain,
			target_host,
			target_uri,
			http_method,
			headers_json
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`
		for _, e := range events {
			headers := e.Headers
			if headers == nil {
				headers = make(map[string][]string)
			}
			headersJSON, err := json.Marshal(headers)
			if err != nil {
				return fmt.Errorf("marshal headers_json: %w", err)
			}
			if _, err := tx.db.ExecContext(
				ctx,
				query,
				e.ClientIP,
				e.DeviceID,
				e.AddressID,
				e.Outcome,
				e.DenyReason,
				e.CreatedAt,
				e.XFFChain,
				e.TargetHost,
				e.TargetURI,
				e.HTTPMethod,
				string(headersJSON),
			); err != nil {
				return fmt.Errorf("insert audit event: %w", err)
			}
		}
		return nil
	})
}

// RunInTx runs the callback function inside a transaction.
// If already running in a transaction context, do not create a new one and reuse it
func (r *Repository) runInTx(ctx context.Context, fn func(*Repository) error) error {
	logger := slog.Default()
	if r.rootDB == nil {
		// We are already in a transaction. Do not nest it.
		return fn(r)
	}

	// Start the transaction
	tx, err := r.rootDB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	// Defer rollback (standard practice)
	defer func() {
		//nolint:staticcheck // Empty branch is intentional - ErrTxDone is expected after commit
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			// Rollback error is only significant if transaction wasn't already committed/rolled back
			logger.Error("failed to rollback transaction", slog.Any(logging.AttrKeyError, err))
		}
	}()

	// Create a COPY of the repository
	// We replace 'db' with the transaction 'tx' and set the rootDB to nil so that it is not reused
	txRepo := &Repository{
		rootDB: nil, // Prevent nested transactions
		db:     tx,  // All queries using txRepo.dbtmp will now use this transaction
	}

	// Run the business logic with the transactional repo
	if err := fn(txRepo); err != nil {
		return err
	}

	// Commit if successful
	return tx.Commit()
}

// RunInTx runs the callback function inside a transaction.
// If already running in a transaction context, do not create a new one and reuse it
func (r *Repository) RunInTx(ctx context.Context, fn func(repository) error) error {
	return r.runInTx(ctx, func(repo *Repository) error {
		return fn(repo)
	})
}
