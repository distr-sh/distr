package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/distr-sh/distr/internal/buildconfig"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/registry/upstream"
	"github.com/distr-sh/distr/internal/svc"
	"github.com/distr-sh/distr/internal/util"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func NewMaintenanceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "maintenance",
		Short: "run maintenance tasks",
	}
	cmd.AddCommand(NewSyncArtifactsUpstreamCommand())
	return cmd
}

type SyncArtifactsUpstreamOptions struct {
	Timeout time.Duration
}

func NewSyncArtifactsUpstreamCommand() *cobra.Command {
	var opts SyncArtifactsUpstreamOptions
	cmd := &cobra.Command{
		Use:    "sync-artifacts-upstream",
		Short:  "sync artifact tags from upstream registries",
		Args:   cobra.NoArgs,
		PreRun: func(cmd *cobra.Command, args []string) { env.Initialize() },
		Run: func(cmd *cobra.Command, args []string) {
			if err := runSyncArtifactsUpstream(cmd.Context(), opts); err != nil {
				os.Exit(1)
			}
		},
	}
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 0, "timeout for the sync operation. 0 means no timeout (default)")
	return cmd
}

func init() {
	RootCommand.AddCommand(NewMaintenanceCommand())
}

func runSyncArtifactsUpstream(ctx context.Context, opts SyncArtifactsUpstreamOptions) error {
	registry := util.Require(svc.NewDefault(ctx))
	defer func() { util.Must(registry.Shutdown(ctx)) }()
	log := registry.GetLogger()

	ctx, _ = signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	ctx = internalctx.WithDb(ctx, registry.GetDbPool())
	ctx = internalctx.WithLogger(ctx, log)
	if s3Client := registry.GetS3Client(); s3Client != nil {
		ctx = internalctx.WithS3Client(ctx, s3Client)
	}

	ctx, span := registry.GetTracers().Always().
		Tracer("github.com/distr-sh/distr/cmd/hub/cmd", trace.WithInstrumentationVersion(buildconfig.Version())).
		Start(ctx, "maintenance_sync-artifacts-upstream", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	log.Info("starting upstream sync", zap.Duration("timeout", opts.Timeout))

	if err := upstream.RunUpstreamSync(ctx, true); err != nil {
		log.Error("upstream sync failed", zap.Error(err))
		span.SetStatus(codes.Error, "upstream sync error")
		span.RecordError(err)
		return err
	}
	span.SetStatus(codes.Ok, "upstream sync finished")
	return nil
}
