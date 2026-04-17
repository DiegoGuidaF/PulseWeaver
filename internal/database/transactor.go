package database

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// Transactor is the service-facing handle. It exposes ONLY tx orchestration —
// no query methods. A service holding one can compose operations atomically
// but cannot perform SQL.
type Transactor struct {
	pool *sqlx.DB
}

func NewTransactor(pool *sqlx.DB) *Transactor {
	return &Transactor{pool: pool}
}

func (t *Transactor) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return withinTx(ctx, t.pool, fn)
}
