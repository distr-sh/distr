package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/distr-sh/distr/internal/buildconfig"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
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
	cmd.AddCommand(NewVerifyCustomDomainsCommand())
	return cmd
}

func newMaintenanceTaskCommand(
	use, short string,
	run func(ctx context.Context) error,
) *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:    use,
		Short:  short,
		Args:   cobra.NoArgs,
		PreRun: func(cmd *cobra.Command, args []string) { env.Initialize() },
		Run: func(cmd *cobra.Command, args []string) {
			if err := runMaintenanceTask(cmd.Context(), use, timeout, run); err != nil {
				os.Exit(1)
			}
		},
	}
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "timeout for the operation. 0 means no timeout (default)")
	return cmd
}

func NewSyncArtifactsUpstreamCommand() *cobra.Command {
	return newMaintenanceTaskCommand(
		"sync-artifacts-upstream",
		"sync artifact tags from upstream registries",
		func(ctx context.Context) error { return upstream.RunUpstreamSync(ctx, true) },
	)
}

func NewVerifyCustomDomainsCommand() *cobra.Command {
	return newMaintenanceTaskCommand(
		"verify-custom-domains",
		"check the CNAME records of custom domains",
		customdomains.RunCustomDomainVerification,
	)
}

func init() {
	RootCommand.AddCommand(NewMaintenanceCommand())
}

func runMaintenanceTask(
	ctx context.Context,
	name string,
	timeout time.Duration,
	run func(ctx context.Context) error,
) error {
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
		Start(ctx, "maintenance_"+name, trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	log.Info("starting maintenance task", zap.String("task", name), zap.Duration("timeout", timeout))

	if err := run(ctx); err != nil {
		log.Error("maintenance task failed", zap.String("task", name), zap.Error(err))
		span.SetStatus(codes.Error, "maintenance task error")
		span.RecordError(err)
		return err
	}
	span.SetStatus(codes.Ok, "maintenance task finished")
	return nil
}
