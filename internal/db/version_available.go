package db

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetDeploymentsPendingUpdate returns the deployments of the given version's application that run
// a different version of it and whose entitlement, if they were created with one, covers the given
// version. Whether that version is older is left to the caller, since the ordering depends on the
// application's versioning strategy.
func GetDeploymentsPendingUpdate(
	ctx context.Context,
	applicationVersionID uuid.UUID,
) ([]types.DeploymentPendingUpdate, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT
			d.id AS deployment_id,
			d.release_name,
			dt.id AS deployment_target_id,
			dt.name AS deployment_target_name,
			dt.customer_organization_id,
			co.name AS customer_organization_name,
			co.partner_organization_id,
			av.id AS current_version_id,
			av.name AS current_version_name
		FROM Deployment d
			-- The latest revision rather than the current one: an automatic update has already
			-- created a revision for the new version, and that deployment needs no notification.
			JOIN DeploymentRevision dr ON dr.id = d.latest_deployment_revision_id
			JOIN ApplicationVersion av ON av.id = dr.application_version_id
			JOIN DeploymentTarget dt ON dt.id = d.deployment_target_id
			LEFT JOIN CustomerOrganization co ON co.id = dt.customer_organization_id
		WHERE av.application_id = (
				SELECT application_id FROM ApplicationVersion WHERE id = @applicationVersionID
			)
			AND dr.application_version_id <> @applicationVersionID
			AND (d.application_entitlement_id IS NULL OR EXISTS (
				SELECT 1 FROM ApplicationEntitlement ae
				WHERE ae.id = d.application_entitlement_id
					AND (ae.expires_at IS NULL OR ae.expires_at > now())
					AND (
						NOT EXISTS (
							SELECT 1 FROM ApplicationEntitlement_ApplicationVersion aeav
							WHERE aeav.application_entitlement_id = ae.id
						)
						OR EXISTS (
							SELECT 1 FROM ApplicationEntitlement_ApplicationVersion aeav
							WHERE aeav.application_entitlement_id = ae.id
								AND aeav.application_version_id = @applicationVersionID
						)
					)
			))
		ORDER BY co.name, dt.name`,
		pgx.NamedArgs{"applicationVersionID": applicationVersionID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query deployments pending update: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[types.DeploymentPendingUpdate])
}

// GetArtifactVersionEntitlement returns which customer organizations may know that the given
// artifact version exists. An entitlement for the whole artifact covers every version, and one
// for a version that resolves to the same content covers it too, be that a sibling tag or an index
// the version is part of.
func GetArtifactVersionEntitlement(
	ctx context.Context,
	artifactID, artifactVersionID uuid.UUID,
) (types.GetArtifactVersionEntitlementResult, error) {
	db := internalctx.GetDb(ctx)

	var result types.GetArtifactVersionEntitlementResult
	gatedRows, err := db.Query(
		ctx,
		`SELECT EXISTS (
			SELECT 1 FROM ArtifactEntitlement ae
			JOIN Artifact a ON a.organization_id = ae.organization_id
			WHERE a.id = @artifactID
		)`,
		pgx.NamedArgs{"artifactID": artifactID},
	)
	if err != nil {
		return result, fmt.Errorf("failed to query artifact entitlement existence: %w", err)
	}
	if result.Gated, err = pgx.CollectExactlyOneRow(gatedRows, pgx.RowTo[bool]); err != nil {
		return result, fmt.Errorf("failed to collect artifact entitlement existence: %w", err)
	} else if !result.Gated {
		return result, nil
	}

	rows, err := db.Query(
		ctx,
		`WITH RECURSIVE equivalent (id, manifest_blob_digest) AS (
			SELECT av.id, av.manifest_blob_digest
			FROM ArtifactVersion av
			WHERE av.artifact_id = @artifactID
				AND av.manifest_blob_digest = (
					SELECT manifest_blob_digest FROM ArtifactVersion WHERE id = @artifactVersionID
				)

			UNION

			SELECT av.id, av.manifest_blob_digest
			FROM ArtifactVersion av
			JOIN ArtifactVersionPart avp ON avp.artifact_version_id = av.id
			JOIN equivalent e ON avp.artifact_blob_digest = e.manifest_blob_digest
		)
		SELECT DISTINCT ae.customer_organization_id
		FROM ArtifactEntitlement ae
		JOIN ArtifactEntitlement_Artifact aea ON aea.artifact_entitlement_id = ae.id
		WHERE aea.artifact_id = @artifactID
			AND ae.customer_organization_id IS NOT NULL
			AND (ae.expires_at IS NULL OR ae.expires_at > now())
			AND (
				aea.artifact_version_id IS NULL
				OR aea.artifact_version_id IN (SELECT id FROM equivalent)
			)`,
		pgx.NamedArgs{"artifactID": artifactID, "artifactVersionID": artifactVersionID},
	)
	if err != nil {
		return result, fmt.Errorf("failed to query entitled customer organizations: %w", err)
	}
	if result.CustomerOrganizationIDs, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID]); err != nil {
		return result, fmt.Errorf("failed to collect entitled customer organizations: %w", err)
	}

	return result, nil
}
