package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/svc"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func main() {
	ctx := context.Background()
	env.Initialize()
	registry := util.Require(svc.NewDefault(ctx))
	defer func() { _ = registry.Shutdown(ctx) }()
	ctx = internalctx.WithDb(ctx, registry.GetDbPool())
	db := internalctx.GetDb(ctx)

	orgID := uuid.MustParse("ff6b33e1-9c1b-495c-bc56-fe9fadf44dc1") // pmig (Enterprise)

	// Fetch artifact version IDs for this org
	rows, err := db.Query(ctx,
		`SELECT av.id FROM ArtifactVersion av
		 JOIN Artifact a ON a.id = av.artifact_id
		 WHERE a.organization_id = $1`, orgID)
	util.Must(err)
	versionIDs := util.Require(pgx.CollectRows(rows, pgx.RowTo[uuid.UUID]))
	fmt.Printf("Found %d artifact versions\n", len(versionIDs))

	// Fetch user IDs that have pulled from this org before
	rows, err = db.Query(ctx,
		`SELECT DISTINCT ua.id FROM UserAccount ua
		 JOIN ArtifactVersionPull avp ON avp.useraccount_id = ua.id
		 JOIN ArtifactVersion av ON av.id = avp.artifact_version_id
		 JOIN Artifact a ON a.id = av.artifact_id
		 WHERE a.organization_id = $1`, orgID)
	util.Must(err)
	userIDs := util.Require(pgx.CollectRows(rows, pgx.RowTo[uuid.UUID]))
	fmt.Printf("Found %d users\n", len(userIDs))

	// Fetch customer org IDs for this org
	rows, err = db.Query(ctx,
		`SELECT id FROM CustomerOrganization WHERE organization_id = $1`, orgID)
	util.Must(err)
	customerOrgIDs := util.Require(pgx.CollectRows(rows, pgx.RowTo[uuid.UUID]))
	fmt.Printf("Found %d customer organizations\n", len(customerOrgIDs))

	if len(versionIDs) == 0 || len(userIDs) == 0 || len(customerOrgIDs) == 0 {
		panic("not enough data to generate pulls")
	}

	remoteAddresses := []string{
		"192.168.1.10", "192.168.1.20", "10.0.0.5", "10.0.0.42",
		"172.16.0.100", "172.16.0.200", "203.0.113.10", "203.0.113.50",
		"198.51.100.1", "198.51.100.99",
	}

	totalCount := 1_000_000
	batchSize := 50_000
	now := time.Now().UTC()
	// Spread pulls over the last 180 days
	startTime := now.AddDate(0, 0, -180)

	inserted := 0
	for inserted < totalCount {
		remaining := totalCount - inserted
		currentBatch := batchSize
		if remaining < currentBatch {
			currentBatch = remaining
		}

		count, err := db.CopyFrom(
			ctx,
			pgx.Identifier{"artifactversionpull"},
			[]string{"created_at", "artifact_version_id", "useraccount_id", "remote_address", "customer_organization_id"},
			pgx.CopyFromSlice(currentBatch, func(i int) ([]any, error) {
				// Random timestamp within the window
				offset := time.Duration(rand.Int63n(int64(now.Sub(startTime))))
				createdAt := startTime.Add(offset)

				versionID := versionIDs[rand.Intn(len(versionIDs))]
				userID := userIDs[rand.Intn(len(userIDs))]
				addr := remoteAddresses[rand.Intn(len(remoteAddresses))]

				// ~80% of pulls have a customer org, ~20% are nil
				var customerOrgID *uuid.UUID
				if rand.Float32() < 0.8 {
					id := customerOrgIDs[rand.Intn(len(customerOrgIDs))]
					customerOrgID = &id
				}

				return []any{createdAt, versionID, userID, &addr, customerOrgID}, nil
			}),
		)
		util.Must(err)

		inserted += int(count)
		fmt.Printf("Inserted %d / %d pulls\n", inserted, totalCount)
	}

	fmt.Println("Done!")
}
