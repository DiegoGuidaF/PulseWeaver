package testutils

import "context"

type NoopTransactor struct{}

func (NoopTransactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
