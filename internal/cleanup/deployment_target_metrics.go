package cleanup

import (
	"context"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"go.uber.org/zap"
)

func RunDeploymentTargetMetricsCleanup(ctx context.Context) error {
	log := internalctx.GetLogger(ctx)
	if count, err := db.CleanupDeploymentTargetMetrics(ctx); err != nil {
		return err
	} else {
		log.Info("DeploymentTargetMetrics cleanup finished", zap.Int64("rowsDeleted", count))
	}
	if count, err := db.CleanupDeploymentMetrics(ctx); err != nil {
		return err
	} else {
		log.Info("DeploymentMetrics cleanup finished", zap.Int64("rowsDeleted", count))
		return nil
	}
}
