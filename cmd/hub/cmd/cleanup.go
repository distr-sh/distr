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
	deploymentTargetMetrics   = "DeploymentTargetMetrics"
	deploymentRevisionStatus  = "DeploymentRevisionStatus"
	deploymentLogRecord       = "DeploymentLogRecord"
	deploymentTargetLogRecord = "DeploymentTargetLogRecord"
	oidcState                 = "OIDCState"
	artifactBlob              = "ArtifactBlob"
)

type CleanupOptions struct {
	Type    string
	Timeout time.Duration
}

func NewCleanupCommand() *cobra.Command {
	var opts CleanupOptions
	cmd := cobra.Command{
		Use: "cleanup <type>",
		Long: fmt.Sprintf(
			"type must be one of: %v, %v, %v, %v, %v, %v",
			deploymentRevisionStatus,
			deploymentTargetMetrics,
			deploymentLogRecord,
			deploymentTargetLogRecord,
			oidcState,
			artifactBlob,
		),
		Short: "delete old data",
		Args:  cobra.ExactArgs(1),
		ValidArgs: []cobra.Completion{
			deploymentRevisionStatus,
			deploymentTargetMetrics,
			deploymentLogRecord,
			deploymentTargetLogRecord,
			oidcState,
			artifactBlob,
		},
		PreRun: func(cmd *cobra.Command, args []string) { env.Initialize() },
		Run: func(cmd *cobra.Command, args []string) {
			opts.Type = args[0]
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

func runCleanup(ctx context.Context, opts CleanupOptions) error {
	registry := util.Require(svc.NewDefault(ctx))
	defer func() { util.Must(registry.Shutdown(ctx)) }()
	log := registry.GetLogger()

	var cleanupFunc func(context.Context) error
	switch opts.Type {
	case deploymentRevisionStatus:
		cleanupFunc = cleanup.RunDeploymentRevisionStatusCleanup
	case deploymentTargetMetrics:
		cleanupFunc = cleanup.RunDeploymentTargetMetricsCleanup
	case deploymentLogRecord:
		cleanupFunc = cleanup.RunDeploymentLogRecordCleanup
	case deploymentTargetLogRecord:
		cleanupFunc = cleanup.RunDeploymentTargetLogRecordCleanup
	case oidcState:
		cleanupFunc = cleanup.RunOIDCStateCleanup
	case artifactBlob:
		cleanupFunc = cleanup.RunArtifactBlobCleanup
	default:
		log.Sugar().Errorf("invalid cleanup type: %v", opts.Type)
		return errors.New("invalid cleanup type")
	}

	ctx, _ = signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	ctx = internalctx.WithDb(ctx, registry.GetDbPool())
	ctx = internalctx.WithLogger(ctx, log)

	ctx, span := registry.GetTracers().Always().
		Tracer("github.com/distr-sh/distr/cmd/hub/cmd", trace.WithInstrumentationVersion(buildconfig.Version())).
		Start(ctx, fmt.Sprintf("cleanup_%v", opts.Type), trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	log.Info("starting cleanup", zap.String("type", opts.Type), zap.Duration("timeout", opts.Timeout))

	if err := cleanupFunc(ctx); err != nil {
		log.Error("cleanup failed", zap.Error(err))
		span.SetStatus(codes.Error, "cleanupFunc error")
		span.RecordError(err)
		return err
	}
	span.SetStatus(codes.Ok, "cleanupFunc finished")
	return nil
}
