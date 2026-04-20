package context

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/distr-sh/distr/internal/db/queryable"
	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/prometheus"
	"github.com/go-mailx/mailx"
	"go.uber.org/zap"
)

type contextKey int

const (
	ctxKeyDb contextKey = iota
	ctxKeyLogger
	ctxKeyMailer
	ctxKeyOrgId
	ctxKeyApplication
	ctxKeyArtifact
	ctxKeyDeployment
	ctxKeyDeploymentTarget
	ctxKeyFile
	ctxKeyUserAccount
	ctxKeyApplicationEntitlement
	ctxKeyArtifactEntitlement
	ctxKeyIPAddress
	ctxKeyOIDCer
	ctxKeyLicenseKey
	ctxKeyPrometheusCollector
	ctxKeyS3Client
)

func GetDb(ctx context.Context) queryable.Queryable {
	val := ctx.Value(ctxKeyDb)
	if db, ok := val.(queryable.Queryable); ok {
		if db != nil {
			return db
		}
	}
	panic("db not contained in context")
}

func WithDb(ctx context.Context, db queryable.Queryable) context.Context {
	ctx = context.WithValue(ctx, ctxKeyDb, db)
	return ctx
}

func GetLogger(ctx context.Context) *zap.Logger {
	val := ctx.Value(ctxKeyLogger)
	if logger, ok := val.(*zap.Logger); ok {
		if logger != nil {
			return logger
		}
	}
	panic("logger not contained in context")
}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	ctx = context.WithValue(ctx, ctxKeyLogger, logger)
	return ctx
}

func GetMailer(ctx context.Context) *mailx.Mailer {
	if mailer, ok := ctx.Value(ctxKeyMailer).(*mailx.Mailer); ok {
		if mailer != nil {
			return mailer
		}
	}
	panic("mailer not contained in context")
}

func WithMailer(ctx context.Context, mailer *mailx.Mailer) context.Context {
	return context.WithValue(ctx, ctxKeyMailer, mailer)
}

func GetOIDCer(ctx context.Context) *oidc.OIDCer {
	if oidcer, ok := ctx.Value(ctxKeyOIDCer).(*oidc.OIDCer); ok {
		if oidcer != nil {
			return oidcer
		}
	}
	panic("oidcer not contained in context")
}

func WithOIDCer(ctx context.Context, oidcer *oidc.OIDCer) context.Context {
	return context.WithValue(ctx, ctxKeyOIDCer, oidcer)
}

// GetPrometheusCollector returns the Prometheus collector from the context.
//
// Important: In contrast to other context accessors, GetPrometheusCollector does not panic if the collector is not
// found. Instead, it returns nil.
func GetPrometheusCollector(ctx context.Context) *prometheus.DistrCollector {
	if collector, ok := ctx.Value(ctxKeyPrometheusCollector).(*prometheus.DistrCollector); ok {
		return collector
	}
	return nil
}

func WithPrometheusCollector(ctx context.Context, collector *prometheus.DistrCollector) context.Context {
	if collector == nil {
		return ctx
	}

	return context.WithValue(ctx, ctxKeyPrometheusCollector, collector)
}

func GetS3Client(ctx context.Context) *s3.Client {
	if client, ok := ctx.Value(ctxKeyS3Client).(*s3.Client); ok {
		if client != nil {
			return client
		}
	}
	panic("s3 client not contained in context")
}

func WithS3Client(ctx context.Context, client *s3.Client) context.Context {
	return context.WithValue(ctx, ctxKeyS3Client, client)
}
