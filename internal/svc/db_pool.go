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

func (r *Registry) GetDbReadonlyPool() *pgxpool.Pool {
	return r.dbReadonlyPool
}

func (reg *Registry) createDBPool(ctx context.Context) (*pgxpool.Pool, error) {
	return reg.createDBPoolFor(ctx, env.DatabaseUrl(), env.DatabaseMaxConns())
}

// createDBReadonlyPool creates a connection pool for the optional read-only database. When no
// read-only URL is configured, it returns nil and callers should keep using the primary pool.
func (reg *Registry) createDBReadonlyPool(ctx context.Context) (*pgxpool.Pool, error) {
	readonlyUrl := env.DatabaseReadonlyUrl()
	if readonlyUrl == nil {
		reg.logger.Info("no read-only database configured, using primary db pool for all queries")
		return nil, nil
	}
	reg.logger.Info("setting up read-only db pool")
	return reg.createDBPoolFor(ctx, *readonlyUrl, env.DatabaseReadonlyMaxConns())
}

func (reg *Registry) createDBPoolFor(ctx context.Context, url string, maxConns *int) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(url)
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
	if maxConns != nil {
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
