package cleanup

import (
	"context"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"go.uber.org/zap"
)

func RunOrganizationCleanup(ctx context.Context) error {
	log := internalctx.GetLogger(ctx)
	if count, err := db.DeleteOrganizationsOlderThan(ctx, env.CleanupOrganizationMinAge()); err != nil {
		return err
	} else {
		log.Info("Organization cleanup finished", zap.Int64("rowsDeleted", count))
		return nil
	}
}
