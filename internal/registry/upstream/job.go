package upstream

import (
	"context"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"go.uber.org/zap"
)

func RunUpstreamSync(ctx context.Context, skipExistingTags bool) error {
	log := internalctx.GetLogger(ctx)
	artifacts, err := db.GetArtifactsWithUpstreamURL(ctx)
	if err != nil {
		return err
	}
	syncer := &Syncer{}
	for _, artifact := range artifacts {
		if err := syncer.SyncArtifactTags(ctx, &artifact, skipExistingTags); err != nil {
			log.Warn("upstream sync failed", zap.Stringer("artifactId", artifact.ID), zap.Error(err))
		}
	}
	return nil
}
