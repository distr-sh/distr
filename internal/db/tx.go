package db

import (
	"context"
	"errors"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db/queryable"
	"github.com/jackc/pgx/v5"
	"go.uber.org/multierr"
)

// RunTx runs a transaction with the PostgreSQL default isolation level (ReadCommitted).
func RunTx(ctx context.Context, f func(ctx context.Context) error) (finalErr error) {
	db := internalctx.GetDb(ctx)
	if tx, err := db.Begin(ctx); err != nil {
		return err
	} else {
		return runTxFunc(ctx, tx, f)
	}
}

// RunTxRR runs a transaction with isolation level RepeatableRead.
func RunTxRR(ctx context.Context, f func(ctx context.Context) error) (finalErr error) {
	return RunTxIso(ctx, pgx.RepeatableRead, f)
}

// RunTxIso runs a transaction with the specified isolation level.
func RunTxIso(ctx context.Context, isoLevel pgx.TxIsoLevel, f func(ctx context.Context) error) error {
	db := internalctx.GetDb(ctx)
	if conn, ok := db.(queryable.Conn); ok {
		if tx, err := conn.BeginEx(ctx, &pgx.TxOptions{IsoLevel: isoLevel}); err != nil {
			return err
		} else {
			return runTxFunc(ctx, tx, f)
		}
	} else if conn, ok := db.(queryable.PoolConn); ok {
		if tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: isoLevel}); err != nil {
			return err
		} else {
			return runTxFunc(ctx, tx, f)
		}
	} else {
		return errors.New("RunTxIso can not be called from within an existing transaction")
	}
}

func runTxFunc(ctx context.Context, tx pgx.Tx, f func(ctx context.Context) error) (finalErr error) {
	defer func() {
		// Rollback is safe to call after commit but we have to silence ErrTxClosed
		if err := tx.Rollback(ctx); !errors.Is(err, pgx.ErrTxClosed) {
			multierr.AppendInto(&finalErr, err)
		}
	}()
	txa := WithAfterFunc{Queryable: tx}
	if err := f(internalctx.WithDb(ctx, &txa)); err != nil {
		return err
	} else {
		if err := tx.Commit(ctx); err != nil {
			return err
		} else {
			for _, f := range txa.AfterFunc {
				f(ctx)
			}
		}
		return nil
	}
}

type WithAfterFunc struct {
	queryable.Queryable
	AfterFunc []func(context.Context)
}

// RunAfterTx runs a function after the transaction is committed.
// If the Queryable is not a [WithAfterFunc], the function is run immediately.
func RunAfterTx(ctx context.Context, f func(context.Context)) {
	db := internalctx.GetDb(ctx)
	if tx, ok := db.(*WithAfterFunc); ok {
		tx.AfterFunc = append(tx.AfterFunc, f)
	} else {
		f(ctx)
	}
}
