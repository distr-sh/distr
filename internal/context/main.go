package context

import (
	"context"

	"github.com/distr-sh/distr/internal/db/queryable"
	"github.com/distr-sh/distr/internal/mail"
	"github.com/distr-sh/distr/internal/oidc"
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
	ctxKeyApplicationLicense
	ctxKeyArtifactLicense
	ctxKeyIPAddress
	ctxKeyOIDCer
	ctxKeyUsageLicense
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

func GetMailer(ctx context.Context) mail.Mailer {
	if mailer, ok := ctx.Value(ctxKeyMailer).(mail.Mailer); ok {
		if mailer != nil {
			return mailer
		}
	}
	panic("mailer not contained in context")
}

func WithMailer(ctx context.Context, mailer mail.Mailer) context.Context {
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
