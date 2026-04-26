package database

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

// DB is the repo-facing handle. It has the full query surface + WithinTx.
type DB struct {
	pool *sqlx.DB
}

func newDB(pool *sqlx.DB) *DB { return &DB{pool: pool} }

func (d *DB) exec(ctx context.Context) sqlx.ExtContext {
	if tx, ok := ctx.Value(txCtxKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return d.pool
}

// WithinTx available on DB so repos can scope their OWN atomic flows.
func (d *DB) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return withinTx(ctx, d.pool, fn)
}

// WithinTx runs fn inside a transaction. If ctx already carries a tx, fn is
// run with that tx — no new tx is started (savepoints are deliberately not used).
func withinTx(ctx context.Context, pool *sqlx.DB, fn func(ctx context.Context) error) (err error) {
	if _, ok := txFromCtx(ctx); ok {
		return fn(ctx)
	}

	tx, err := pool.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				slog.Default().Error("tx rollback failed", slog.Any("error", rbErr))
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			err = commitErr
		}
	}()

	return fn(withTx(ctx, tx))
}

// ─── Query surface ──────────────────────────────────────────────────────────

func (d *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.exec(ctx).ExecContext(ctx, query, args...)
}

func (d *DB) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	return d.exec(ctx).QueryxContext(ctx, query, args...)
}

func (d *DB) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	return d.exec(ctx).QueryRowxContext(ctx, query, args...)
}

func (d *DB) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	return sqlx.GetContext(ctx, d.exec(ctx), dest, query, args...)
}

func (d *DB) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	return sqlx.SelectContext(ctx, d.exec(ctx), dest, query, args...)
}

// Rebind translates `?` placeholders to whatever the underlying driver expects.
// Lets callers compose queries via sqlx.In (e.g. `WHERE id IN (?)`) without
// having to think about driver-specific bind styles.
func (d *DB) Rebind(query string) string {
	return d.pool.Rebind(query)
}

func (d *DB) NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error) {
	q, args, err := sqlx.Named(query, arg)
	if err != nil {
		return nil, err
	}
	ext := d.exec(ctx)
	return ext.ExecContext(ctx, ext.Rebind(q), args...)
}
