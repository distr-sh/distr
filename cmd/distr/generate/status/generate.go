package main

import (
	"context"
	"time"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/svc"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
)

func main() {
	ctx := context.Background()
	env.Initialize()
	registry := util.Require(svc.NewDefault(ctx))
	defer func() { _ = registry.Shutdown(ctx) }()
	ctx = internalctx.WithDb(ctx, registry.GetDbPool())

	revisionID := uuid.MustParse("fb3e0293-a782-4088-a50c-ec43bee8f03d")
	statusCount := 2000000
	statusInterval := 5 * time.Second

	now := time.Now().UTC()
	createdAt := now.Add(time.Duration(statusCount) * -statusInterval)
	var ds []types.DeploymentRevisionStatus
	for createdAt.Before(now) {
		ds = append(ds, types.DeploymentRevisionStatus{CreatedAt: createdAt, Message: "demo status"})
		createdAt = createdAt.Add(statusInterval)
	}
	util.Must(db.BulkCreateDeploymentRevisionStatusWithCreatedAt(ctx, revisionID, ds))
}
