package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	artifactEntitlementOutExpr = `al.id, al.created_at, al.name, al.expires_at, ` +
		`al.customer_organization_id, al.organization_id `
	artifactSelectionsOutExpor = `
		(
			SELECT array_agg(DISTINCT row(
				ala.artifact_id,
				coalesce((
					SELECT array_agg(alax.artifact_version_id) FILTER (WHERE alax.artifact_version_id IS NOT NULL)
					FROM ArtifactEntitlement_Artifact alax
					WHERE alax.artifact_entitlement_id = ala.artifact_entitlement_id AND alax.artifact_id = ala.artifact_id
				 ), ARRAY[]::UUID[])
				))
			FROM ArtifactEntitlement_Artifact ala
			WHERE ala.artifact_entitlement_id = al.id
		) as artifacts `
)

func GetArtifactEntitlements(ctx context.Context, orgID uuid.UUID) ([]types.ArtifactEntitlement, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT `+artifactEntitlementOutExpr+`, `+artifactSelectionsOutExpor+`
		FROM ArtifactEntitlement al
		WHERE al.organization_id = @orgId
		ORDER BY al.name`,
		pgx.NamedArgs{"orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query ArtifactEntitlement: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ArtifactEntitlement])
	if err != nil {
		return nil, fmt.Errorf("could not query ArtifactEntitlement: %w", err)
	}
	return result, nil
}

func CreateArtifactEntitlement(ctx context.Context, entitlement *types.ArtifactEntitlementBase) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		WITH inserted AS (
			INSERT INTO ArtifactEntitlement (
				name, expires_at, organization_id, customer_organization_id
			) VALUES (
				@name, @expiresAt, @organizationId, @customerOrganizationId
			) RETURNING *
		)
		SELECT `+artifactEntitlementOutExpr+`
		FROM inserted al`,
		pgx.NamedArgs{
			"name":                   entitlement.Name,
			"expiresAt":              entitlement.ExpiresAt,
			"organizationId":         entitlement.OrganizationID,
			"customerOrganizationId": entitlement.CustomerOrganizationID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert ArtifactEntitlement: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ArtifactEntitlementBase]); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.UniqueViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return err
	} else {
		*entitlement = result
		return nil
	}
}

func UpdateArtifactEntitlement(ctx context.Context, entitlement *types.ArtifactEntitlementBase) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		WITH updated AS (
			UPDATE ArtifactEntitlement SET
			name = @name,
            expires_at = @expiresAt,
            customer_organization_id = @customerOrganizationId
		 	WHERE id = @id AND organization_id = @organizationId RETURNING *
		)
		SELECT `+artifactEntitlementOutExpr+`
		FROM updated al`,
		pgx.NamedArgs{
			"id":                     entitlement.ID,
			"organizationId":         entitlement.OrganizationID,
			"name":                   entitlement.Name,
			"expiresAt":              entitlement.ExpiresAt,
			"customerOrganizationId": entitlement.CustomerOrganizationID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not update ArtifactEntitlement: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ArtifactEntitlementBase]); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.UniqueViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return err
	} else {
		*entitlement = result
		return nil
	}
}

func RemoveAllArtifactsFromEntitlement(
	ctx context.Context,
	id uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(
		ctx,
		`DELETE FROM ArtifactEntitlement_Artifact
		WHERE artifact_entitlement_id = @artifactEntitlementId`,
		pgx.NamedArgs{
			"artifactEntitlementId": id,
		},
	)
	if err != nil {
		return fmt.Errorf("could not delete relation: %w", err)
	} else {
		return nil
	}
}

func AddArtifactToArtifactEntitlement(
	ctx context.Context,
	entitlementID uuid.UUID,
	artifactId uuid.UUID,
	artifactVersionId *uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(
		ctx,
		`INSERT INTO ArtifactEntitlement_Artifact (artifact_entitlement_id, artifact_id, artifact_version_id)
		VALUES (@entitlementId, @id, @versionId)
		ON CONFLICT (artifact_entitlement_id, artifact_id, artifact_version_id) DO NOTHING`,
		pgx.NamedArgs{
			"entitlementId": entitlementID,
			"id":            artifactId,
			"versionId":     artifactVersionId,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert relation: %w", err)
	}
	return nil
}

func GetArtifactEntitlementByID(ctx context.Context, id uuid.UUID) (*types.ArtifactEntitlement, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
			SELECT `+artifactEntitlementOutExpr+`, `+artifactSelectionsOutExpor+`
			FROM ArtifactEntitlement al
			WHERE al.id = @id `,
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query ArtifactEntitlement: %w", err)
	}

	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ArtifactEntitlement]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not collect ArtifactEntitlement: %w", err)
	} else {
		return &result, nil
	}
}

func DeleteArtifactEntitlementWithID(ctx context.Context, id uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx, `DELETE FROM ArtifactEntitlement WHERE id = @id`, pgx.NamedArgs{"id": id})
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.ForeignKeyViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return err
	} else if cmd.RowsAffected() == 0 {
		err = apierrors.ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("could not delete ArtifactEntitlement: %w", err)
	}

	return nil
}

func DeleteArtifactEntitlementsWithOrganizationID(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(
		ctx,
		`DELETE FROM ArtifactEntitlement WHERE organization_id = @organizationID`,
		pgx.NamedArgs{"organizationID": organizationID},
	)
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.ForeignKeyViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return 0, fmt.Errorf("could not delete ArtifactEntitlement: %w", err)
	}

	return cmd.RowsAffected(), nil
}

func DeleteArtifactEntitlementsWithOrganizationSubscriptionType(
	ctx context.Context,
	subscriptionType []types.SubscriptionType,
) (int64, error) {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(
		ctx,
		`DELETE FROM ArtifactEntitlement WHERE organization_id IN (
			SELECT id FROM Organization WHERE subscription_type = ANY(@subscriptionType)
		)`,
		pgx.NamedArgs{"subscriptionType": subscriptionType},
	)
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.ForeignKeyViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return 0, fmt.Errorf("could not delete ArtifactEntitlements: %w", err)
	}

	return cmd.RowsAffected(), nil
}
