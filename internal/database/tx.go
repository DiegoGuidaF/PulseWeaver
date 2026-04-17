package database

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type txCtxKey struct{}

func Exec(ctx context.Context, db *sqlx.DB) sqlx.ExtContext {
	if tx, ok := ctx.Value(txCtxKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return db
}

func withTx(ctx context.Context, tx *sqlx.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

func txFromCtx(ctx context.Context) (*sqlx.Tx, bool) {
	tx, ok := ctx.Value(txCtxKey{}).(*sqlx.Tx)
	return tx, ok
}
