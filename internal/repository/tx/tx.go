// Package tx is the repository layer's shared transaction primitive. It
// lets a repository method run either directly against the pool or,
// transparently, against a pgx.Tx carried on ctx — so code composing
// several repository calls into one atomic unit doesn't have to thread a
// *pgx.Tx through every method signature it touches.
package tx

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is the subset of *pgxpool.Pool and pgx.Tx that a repository
// method needs — enough to run a query without caring which one it's
// actually talking to.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txKey struct{}

// WithTx runs fn inside a transaction. If ctx already carries one — this
// call is nested inside another WithTx — it reuses that transaction
// instead of starting a new one, so composing WithTx calls (e.g. a
// usecase calling into two repositories that each also use WithTx) never
// nests transactions or silently loses atomicity. Otherwise it begins a
// new one, committing it if fn succeeds and rolling it back otherwise.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(ctx context.Context) error) error {
	if _, ok := fromContext(ctx); ok {
		return fn(ctx)
	}

	dbTx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() { _ = dbTx.Rollback(ctx) }() // no-op once committed

	if err := fn(context.WithValue(ctx, txKey{}, dbTx)); err != nil {
		return err
	}

	if err := dbTx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// QuerierFromContext returns the Querier a repository method should run
// against for ctx — the active transaction if WithTx put one there,
// otherwise pool itself.
func QuerierFromContext(ctx context.Context, pool *pgxpool.Pool) Querier {
	if dbTx, ok := fromContext(ctx); ok {
		return dbTx
	}

	return pool
}

func fromContext(ctx context.Context) (pgx.Tx, bool) {
	dbTx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return dbTx, ok
}
