package svc

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/env"
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type loggingQueryTracer struct {
	log *zap.Logger
}

var _ pgx.QueryTracer = &loggingQueryTracer{}

func (tracer *loggingQueryTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	tracer.log.Debug("executing query", zap.String("sql", data.SQL), zap.Any("args", data.Args))
	return ctx
}

func (tracer *loggingQueryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
}

func (r *Registry) GetDbPool() *pgxpool.Pool {
	return r.dbPool
}

func (reg *Registry) createDBPool(ctx context.Context) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(env.DatabaseUrl())
	if err != nil {
		return nil, err
	}
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		typeNames := []string{
			"DEPLOYMENT_TYPE",
			"USER_ROLE",
			"HELM_CHART_TYPE",
			"DEPLOYMENT_STATUS_TYPE",
			"FEATURE",
			"_FEATURE",
			"TUTORIAL",
			"SUBSCRIPTIONTYPE",
			"_SUBSCRIPTIONTYPE",
			"CUSTOMER_ORGANIZATION_FEATURE",
			"_CUSTOMER_ORGANIZATION_FEATURE",
			"UPSTREAM_AUTH_TYPE",
		}
		for _, typeName := range typeNames {
			if pgType, err := conn.LoadType(ctx, typeName); err != nil {
				return err
			} else {
				conn.TypeMap().RegisterType(pgType)
			}
		}
		return nil
	}
	if maxConns := env.DatabaseMaxConns(); maxConns != nil {
		config.MaxConns = int32(*maxConns)
	}
	if env.EnableQueryLogging() {
		config.ConnConfig.Tracer = &loggingQueryTracer{reg.logger}
	} else {
		config.ConnConfig.Tracer = otelpgx.NewTracer(
			otelpgx.WithTracerProvider(reg.GetTracers().Default()),
		)
	}
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("cannot set up db pool: %w", err)
	} else if conn, err := db.Acquire(ctx); err != nil {
		// this actually checks whether the DB can be connected to
		return nil, fmt.Errorf("cannot acquire connection: %w", err)
	} else {
		conn.Release()
		return db, nil
	}
}
