package handlers

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

// customerArtifactUpdates returns the artifacts the customer organizations pulled, the most recently tagged first,
// each with the release tag it pulled last and the newer one it may pull, if any.
func customerArtifactUpdates(
	ctx context.Context,
	customerOrgIDs []uuid.UUID,
) ([]types.CustomerArtifactUpdate, error) {
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()

	pulls, err := db.GetLatestArtifactPullsOfCustomerOrganizations(ctx, orgID, customerOrgIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest artifact pulls: %w", err)
	} else if len(pulls) == 0 {
		return nil, nil
	}
	var artifactIDs []uuid.UUID
	for _, pull := range pulls {
		if !slices.Contains(artifactIDs, pull.Artifact.ID) {
			artifactIDs = append(artifactIDs, pull.Artifact.ID)
		}
	}

	tags, err := db.ListArtifactReleaseTags(ctx, artifactIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact release tags: %w", err)
	}
	visibility := types.NewArtifactTagVisibility(false, nil)
	if auth.CurrentOrg().HasFeature(types.FeatureLicensing) {
		if visibility, err = db.GetArtifactTagVisibility(ctx, orgID, customerOrgIDs); err != nil {
			return nil, fmt.Errorf("failed to get artifact entitlements: %w", err)
		}
	}

	lastTagged := make(map[uuid.UUID]time.Time, len(artifactIDs))
	for _, tag := range tags {
		if tag.CreatedAt.After(lastTagged[tag.ArtifactID]) {
			lastTagged[tag.ArtifactID] = tag.CreatedAt
		}
	}
	slices.SortStableFunc(pulls, func(a, b types.CustomerArtifactPull) int {
		return cmp.Or(
			lastTagged[b.Artifact.ID].Compare(lastTagged[a.Artifact.ID]),
			cmp.Compare(a.Artifact.Name, b.Artifact.Name),
		)
	})

	return types.AvailableArtifactUpdates(pulls, tags, visibility), nil
}
