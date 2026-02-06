package jobs

import (
	"context"
	"time"

	"github.com/distr-sh/distr/internal/buildconfig"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db/queryable"
	"github.com/distr-sh/distr/internal/mail"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	tracerScope = "github.com/distr-sh/distr/internal/jobs"
)

type runner struct {
	db     queryable.Queryable
	mailer mail.Mailer
	logger *zap.Logger
	tracer trace.Tracer
}

func NewRunner(
	logger *zap.Logger,
	db queryable.Queryable,
	mailer mail.Mailer,
	traceProvider trace.TracerProvider,
) *runner {
	runner := runner{
		db:     db,
		mailer: mailer,
		logger: logger,
		tracer: traceProvider.Tracer(tracerScope, trace.WithInstrumentationVersion(buildconfig.Version())),
	}
	return &runner
}

func (runner *runner) RunJobFunc(job Job) func(ctx context.Context) {
	return func(ctx context.Context) { runner.Run(ctx, job) }
}

func (runner *runner) Run(ctx context.Context, job Job) {
	log := runner.logger.With(zap.String("job", job.name))

	ctx = runner.jobCtx(ctx, job)
	ctx, span := runner.tracer.Start(ctx, job.name, trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	startedAt := time.Now()
	log.Info("job started")

	if job.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, job.timeout)
		defer cancel()
	}

	err := job.Run(ctx)
	elapsed := time.Since(startedAt)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "job error")
		log.Warn("job failed", zap.Duration("elapsed", elapsed), zap.Error(err))
	} else {
		span.SetStatus(codes.Ok, "job finished")
		log.Info("job finished", zap.Duration("elapsed", elapsed))
	}
	// TODO: save result to DB
}

func (runner *runner) jobCtx(ctx context.Context, job Job) context.Context {
	ctx = internalctx.WithLogger(ctx, runner.logger.With(zap.String("job", job.name)))
	ctx = internalctx.WithDb(ctx, runner.db)
	ctx = internalctx.WithMailer(ctx, runner.mailer)
	return ctx
}
