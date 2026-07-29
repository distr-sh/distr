package db

import (
	"context"
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/advisory"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetCustomerView loads everything about a customer organization's entitlements that
// advisory.IsVisibleToCustomer needs to decide visibility.
func GetCustomerView(
	ctx context.Context, orgID, customerOrgID uuid.UUID,
) (advisory.CustomerView, error) {
	var view advisory.CustomerView

	hasApplicationEntitlements, err := HasAnyApplicationEntitlement(ctx, orgID)
	if err != nil {
		return view, err
	}
	hasArtifactEntitlements, err := HasAnyArtifactEntitlement(ctx, orgID)
	if err != nil {
		return view, err
	}
	view.OrgHasApplicationEntitlements = hasApplicationEntitlements
	view.OrgHasArtifactEntitlements = hasArtifactEntitlements

	if view.ApplicationEntitlements, err = getApplicationEntitlementRefs(ctx, orgID, customerOrgID); err != nil {
		return view, err
	}
	if view.ArtifactEntitlements, err = getArtifactEntitlementRefs(ctx, orgID, customerOrgID); err != nil {
		return view, err
	}
	if view.DeployedApplicationVersionIDs, err = getDeployedApplicationVersionIDs(ctx, orgID, customerOrgID); err != nil {
		return view, err
	}
	return view, nil
}

// getDeployedApplicationVersionIDs returns every application version the customer has ever
// deployed. Superseded revisions count too, so that a customer who has already upgraded away
// from an affected version still sees the advisory, which is also how
// GetAdvisoryImpactedDeployments reports them to the vendor.
func getDeployedApplicationVersionIDs(
	ctx context.Context, orgID, customerOrgID uuid.UUID,
) ([]uuid.UUID, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT DISTINCT dr.application_version_id
		FROM DeploymentRevision dr
			JOIN Deployment d ON d.id = dr.deployment_id
			JOIN DeploymentTarget dt ON dt.id = d.deployment_target_id
		WHERE dt.organization_id = @orgId
			AND dt.customer_organization_id = @customerOrgId`,
		pgx.NamedArgs{"orgId": orgID, "customerOrgId": customerOrgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query deployed application versions: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("could not get deployed application versions: %w", err)
	}
	return result, nil
}

type entitlementRefRow struct {
	ParentID   uuid.UUID   `db:"parent_id"`
	ExpiresAt  *time.Time  `db:"expires_at"`
	VersionIDs []uuid.UUID `db:"version_ids"`
}

func (r entitlementRefRow) toRef() advisory.EntitlementRef {
	return advisory.EntitlementRef{
		ParentID:   r.ParentID,
		VersionIDs: r.VersionIDs,
		ExpiresAt:  r.ExpiresAt,
	}
}

// getApplicationEntitlementRefs returns one ref per application entitlement. An entitlement
// without any ApplicationEntitlement_ApplicationVersion rows covers every version of its
// application, which is represented by an empty VersionIDs.
func getApplicationEntitlementRefs(
	ctx context.Context, orgID, customerOrgID uuid.UUID,
) ([]advisory.EntitlementRef, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT ae.application_id AS parent_id, ae.expires_at,
			coalesce((
				SELECT array_agg(aeav.application_version_id)
				FROM ApplicationEntitlement_ApplicationVersion aeav
				WHERE aeav.application_entitlement_id = ae.id
			), ARRAY[]::UUID[]) AS version_ids
		FROM ApplicationEntitlement ae
		WHERE ae.organization_id = @orgId
			AND ae.customer_organization_id = @customerOrgId`,
		pgx.NamedArgs{"orgId": orgID, "customerOrgId": customerOrgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query application entitlements: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[entitlementRefRow])
	if err != nil {
		return nil, fmt.Errorf("could not get application entitlements: %w", err)
	}
	refs := make([]advisory.EntitlementRef, 0, len(result))
	for _, row := range result {
		refs = append(refs, row.toRef())
	}
	return refs, nil
}

// getArtifactEntitlementRefs returns one ref per (entitlement, artifact) pair. A pair with
// any NULL artifact_version_id covers the whole artifact, which is represented by an empty
// VersionIDs.
func getArtifactEntitlementRefs(
	ctx context.Context, orgID, customerOrgID uuid.UUID,
) ([]advisory.EntitlementRef, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT aea.artifact_id AS parent_id, ae.expires_at,
			CASE
				WHEN bool_or(aea.artifact_version_id IS NULL) THEN ARRAY[]::UUID[]
				ELSE array_agg(aea.artifact_version_id)
			END AS version_ids
		FROM ArtifactEntitlement ae
			JOIN ArtifactEntitlement_Artifact aea ON aea.artifact_entitlement_id = ae.id
		WHERE ae.organization_id = @orgId
			AND ae.customer_organization_id = @customerOrgId
		GROUP BY ae.id, ae.expires_at, aea.artifact_id`,
		pgx.NamedArgs{"orgId": orgID, "customerOrgId": customerOrgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query artifact entitlements: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[entitlementRefRow])
	if err != nil {
		return nil, fmt.Errorf("could not get artifact entitlements: %w", err)
	}
	refs := make([]advisory.EntitlementRef, 0, len(result))
	for _, row := range result {
		refs = append(refs, row.toRef())
	}
	return refs, nil
}

// AffectedVersions are the application and artifact versions an advisory affects.
// Only versions with relation 'affected' are included, since fixed versions never grant
// customers visibility.
type AffectedVersions struct {
	ApplicationVersions []advisory.VersionRef
	ArtifactVersions    []advisory.VersionRef
}

type affectedVersionRow struct {
	AdvisoryID uuid.UUID `db:"advisory_id"`
	VersionID  uuid.UUID `db:"version_id"`
	ParentID   uuid.UUID `db:"parent_id"`
}

// GetAffectedVersions loads the affected versions of several advisories at once.
//
// Artifact versions are expanded to every version resolving to the same content: first the
// sibling tags sharing a manifest digest, then recursively the indexes that contain the
// affected manifest as a part. This mirrors CheckEntitlementForArtifact so that visibility
// and the actual pull entitlement agree for multi-arch images.
func GetAffectedVersions(
	ctx context.Context, advisoryIDs []uuid.UUID,
) (map[uuid.UUID]AffectedVersions, error) {
	result := make(map[uuid.UUID]AffectedVersions, len(advisoryIDs))
	if len(advisoryIDs) == 0 {
		return result, nil
	}

	db := internalctx.GetDb(ctx)

	applicationRows, err := db.Query(
		ctx,
		`SELECT vav.advisory_id, av.id AS version_id, av.application_id AS parent_id
		FROM AdvisoryApplicationVersion vav
			JOIN ApplicationVersion av ON av.id = vav.application_version_id
		WHERE vav.advisory_id = any(@advisoryIds)
			AND vav.relation = 'affected'`,
		pgx.NamedArgs{"advisoryIds": advisoryIDs},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query affected application versions: %w", err)
	}
	applicationVersions, err := pgx.CollectRows(applicationRows, pgx.RowToStructByName[affectedVersionRow])
	if err != nil {
		return nil, fmt.Errorf("could not get affected application versions: %w", err)
	}

	// UNION rather than UNION ALL: it deduplicates against all rows produced so far, which
	// terminates the recursion even if artifact parts ever form a cycle.
	artifactRows, err := db.Query(
		ctx,
		`WITH RECURSIVE Affected (advisory_id, id, artifact_id, manifest_blob_digest) AS (
			SELECT vrv.advisory_id, sibling.id, sibling.artifact_id, sibling.manifest_blob_digest
				FROM AdvisoryArtifactVersion vrv
				JOIN ArtifactVersion av ON av.id = vrv.artifact_version_id
				JOIN ArtifactVersion sibling
					ON sibling.artifact_id = av.artifact_id
					AND sibling.manifest_blob_digest = av.manifest_blob_digest
				WHERE vrv.advisory_id = any(@advisoryIds)
					AND vrv.relation = 'affected'
			UNION
			SELECT agg.advisory_id, av.id, av.artifact_id, av.manifest_blob_digest
				FROM ArtifactVersion av
				JOIN ArtifactVersionPart avp ON av.id = avp.artifact_version_id
				JOIN Affected agg ON avp.artifact_blob_digest = agg.manifest_blob_digest
		)
		SELECT DISTINCT advisory_id, id AS version_id, artifact_id AS parent_id
		FROM Affected`,
		pgx.NamedArgs{"advisoryIds": advisoryIDs},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query affected artifact versions: %w", err)
	}
	artifactVersions, err := pgx.CollectRows(artifactRows, pgx.RowToStructByName[affectedVersionRow])
	if err != nil {
		return nil, fmt.Errorf("could not get affected artifact versions: %w", err)
	}

	for _, row := range applicationVersions {
		affected := result[row.AdvisoryID]
		affected.ApplicationVersions = append(
			affected.ApplicationVersions,
			advisory.VersionRef{VersionID: row.VersionID, ParentID: row.ParentID},
		)
		result[row.AdvisoryID] = affected
	}
	for _, row := range artifactVersions {
		affected := result[row.AdvisoryID]
		affected.ArtifactVersions = append(
			affected.ArtifactVersions,
			advisory.VersionRef{VersionID: row.VersionID, ParentID: row.ParentID},
		)
		result[row.AdvisoryID] = affected
	}
	return result, nil
}
