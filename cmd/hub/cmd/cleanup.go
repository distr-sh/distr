package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/cleanup"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/svc"
	"github.com/distr-sh/distr/internal/util"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	deploymentTargetMetrics  = "DeploymentTargetMetrics"
	deploymentRevisionStatus = "DeploymentRevisionStatus"
	oidcState                = "OIDCState"
	artifactBlob             = "ArtifactBlob"
	organization             = "Organization"
)

type CleanupOptions struct {
	Types   []string
	Timeout time.Duration
}

func NewCleanupCommand() *cobra.Command {
	var opts CleanupOptions
	cmd := cobra.Command{
		Use: "cleanup <type> [type...]",
		Long: fmt.Sprintf(
			"type must be one of: %v, %v, %v, %v, %v",
			deploymentRevisionStatus,
			deploymentTargetMetrics,
			oidcState,
			artifactBlob,
			organization,
		),
		Short: "delete old data",
		Args:  cobra.MinimumNArgs(1),
		ValidArgs: []cobra.Completion{
			deploymentRevisionStatus,
			deploymentTargetMetrics,
			oidcState,
			artifactBlob,
			organization,
		},
		PreRun: func(cmd *cobra.Command, args []string) { env.Initialize() },
		Run: func(cmd *cobra.Command, args []string) {
			opts.Types = args
			if err := runCleanup(cmd.Context(), opts); err != nil {
				os.Exit(1)
			}
		},
	}

	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 0, "timeout for the cleanup operation. 0 means no timeout (default)")

	return &cmd
}

func init() {
	RootCommand.AddCommand(NewCleanupCommand())
}

func resolveCleanupFunc(cleanupType string, registry *svc.Registry) (func(context.Context) error, error) {
	switch cleanupType {
	case deploymentRevisionStatus:
		return cleanup.RunDeploymentRevisionStatusCleanup, nil
	case deploymentTargetMetrics:
		return cleanup.RunDeploymentTargetMetricsCleanup, nil
	case oidcState:
		return cleanup.RunOIDCStateCleanup, nil
	case artifactBlob:
		if registry.GetS3Client() == nil {
			return nil, errors.New("S3 client not configured; ensure the registry is enabled and S3 is configured")
		}
		return cleanup.RunArtifactBlobCleanup, nil
	case organization:
		return cleanup.RunOrganizationCleanup, nil
	default:
		return nil, fmt.Errorf("invalid cleanup type: %v", cleanupType)
	}
}

func runCleanup(ctx context.Context, opts CleanupOptions) error {
	registry := util.Require(svc.NewDefault(ctx))
	defer func() { util.Must(registry.Shutdown(ctx)) }()
	log := registry.GetLogger()

	cleanupFuncs := make([]func(context.Context) error, 0, len(opts.Types))
	for _, t := range opts.Types {
		f, err := resolveCleanupFunc(t, registry)
		if err != nil {
			log.Error("failed to resolve cleanup type", zap.String("type", t), zap.Error(err))
			return err
		}
		cleanupFuncs = append(cleanupFuncs, f)
	}

	ctx, _ = signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	ctx = internalctx.WithDb(ctx, registry.GetDbPool())
	ctx = internalctx.WithLogger(ctx, log)
	if s3Client := registry.GetS3Client(); s3Client != nil {
		ctx = internalctx.WithS3Client(ctx, s3Client)
	}

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	tracer := registry.GetTracers().Always().
		Tracer("github.com/distr-sh/distr/cmd/hub/cmd", trace.WithInstrumentationVersion(buildconfig.Version()))

	var errs []error
	for i, cleanupType := range opts.Types {
		log.Info("starting cleanup", zap.String("type", cleanupType), zap.Duration("timeout", opts.Timeout))

		runCtx, span := tracer.Start(ctx, fmt.Sprintf("cleanup_%v", cleanupType), trace.WithSpanKind(trace.SpanKindInternal))
		if err := cleanupFuncs[i](runCtx); err != nil {
			log.Error("cleanup failed", zap.String("type", cleanupType), zap.Error(err))
			span.SetStatus(codes.Error, "cleanupFunc error")
			span.RecordError(err)
			errs = append(errs, err)
		} else {
			span.SetStatus(codes.Ok, "cleanupFunc finished")
		}
		span.End()
	}

	return errors.Join(errs...)
}
