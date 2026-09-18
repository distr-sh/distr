package cleanup

import (
	"context"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"go.uber.org/zap"
)

func RunFileCleanup(ctx context.Context) error {
	log := internalctx.GetLogger(ctx)
	if count, err := db.DeleteUnreferencedFilesOlderThan(ctx, env.CleanupFileMinAge()); err != nil {
		return err
	} else {
		log.Info("File cleanup finished", zap.Int64("rowsDeleted", count))
		return nil
	}
}
