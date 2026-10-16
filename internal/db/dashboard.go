package db

import (
	"context"
	"errors"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func GetLatestPullOfArtifactByCustomerOrganization(
	ctx context.Context,
	artifactID uuid.UUID,
	customerOrganizationID uuid.UUID,
) (string, error) {
	db := internalctx.GetDb(ctx)
	if rows, err := db.Query(ctx, `
		SELECT av.name
		FROM ArtifactVersionPull avpl
		JOIN ArtifactVersion av ON av.id = avpl.artifact_version_id
		WHERE av.artifact_id = @artifactId
			AND `+artifactVersionIsTagExpr("av")+`
			AND `+artifactPullOfCustomerOrgExpr+`
		ORDER BY avpl.created_at DESC
		LIMIT 1;
	`, pgx.NamedArgs{
		"artifactId":             artifactID,
		"customerOrganizationId": customerOrganizationID,
	}); err != nil {
		return "", err
	} else if res, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[string]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", distrerr.ErrNotFound
		}
		return "", err
	} else {
		return res, nil
	}
}
