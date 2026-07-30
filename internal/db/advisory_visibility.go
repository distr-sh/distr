package db

import (
	"context"
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/advisory"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
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

// getCurrentApplicationVersionIDs returns the application versions the deployments in scope
// run right now, one per deployment. This is the counterpart to
// getDeployedApplicationVersionIDs: that one answers "were they ever exposed", which decides
// visibility, and this one answers "are they still exposed", which decides what
// advisory.IsStillAffected reports.
//
// Vendors are not scoped and never ask for this, so an unscoped call returns nothing rather
// than every deployment in the organization.
func getCurrentApplicationVersionIDs(
	ctx context.Context, orgID uuid.UUID, scope AdvisoryScope,
) ([]uuid.UUID, error) {
	if scope.CustomerOrgID == nil && scope.PartnerOrgID == nil {
		return nil, nil
	}
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT DISTINCT ON (d.id) dr.application_version_id
		FROM DeploymentRevision dr
			JOIN Deployment d ON d.id = dr.deployment_id
			JOIN DeploymentTarget dt ON dt.id = d.deployment_target_id
			LEFT JOIN CustomerOrganization co ON co.id = dt.customer_organization_id
		WHERE dt.organization_id = @orgId
			AND (@partnerOrgId::uuid IS NULL OR co.partner_organization_id = @partnerOrgId)
			AND (@customerOrgId::uuid IS NULL OR dt.customer_organization_id = @customerOrgId)
		ORDER BY d.id, dr.created_at DESC`,
		scope.bind(pgx.NamedArgs{"orgId": orgID}),
	)
	if err != nil {
		return nil, fmt.Errorf("could not query current application versions: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("could not get current application versions: %w", err)
	}
	return result, nil
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

type affectedVersionRow struct {
	AdvisoryID uuid.UUID `db:"advisory_id"`
	VersionID  uuid.UUID `db:"version_id"`
	ParentID   uuid.UUID `db:"parent_id"`
}

type markedVersionRow struct {
	affectedVersionRow
	Relation types.AdvisoryVersionRelation `db:"relation"`
}

// GetMarkedVersions loads the versions several advisories mark, all at once.
//
// Artifact versions are expanded to every version resolving to the same content: first the
// sibling tags sharing a manifest digest, then recursively the indexes that contain the
// marked manifest as a part. This mirrors CheckEntitlementForArtifact so that visibility
// and the actual pull entitlement agree for multi-arch images. It matters just as much for
// the fixed side, where a customer who pulled the fix by tag must be recognised as having
// taken it.
func GetMarkedVersions(
	ctx context.Context, advisoryIDs []uuid.UUID,
) (map[uuid.UUID]advisory.MarkedVersions, error) {
	result := make(map[uuid.UUID]advisory.MarkedVersions, len(advisoryIDs))
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
	// terminates the recursion even if artifact parts ever form a cycle. The relation travels
	// with each row so that both sides come back from one traversal.
	artifactRows, err := db.Query(
		ctx,
		`WITH RECURSIVE Marked (advisory_id, relation, id, artifact_id, manifest_blob_digest) AS (
			SELECT vrv.advisory_id, vrv.relation, sibling.id, sibling.artifact_id,
					sibling.manifest_blob_digest
				FROM AdvisoryArtifactVersion vrv
				JOIN ArtifactVersion av ON av.id = vrv.artifact_version_id
				JOIN ArtifactVersion sibling
					ON sibling.artifact_id = av.artifact_id
					AND sibling.manifest_blob_digest = av.manifest_blob_digest
				WHERE vrv.advisory_id = any(@advisoryIds)
			UNION
			SELECT agg.advisory_id, agg.relation, av.id, av.artifact_id, av.manifest_blob_digest
				FROM ArtifactVersion av
				JOIN ArtifactVersionPart avp ON av.id = avp.artifact_version_id
				JOIN Marked agg ON avp.artifact_blob_digest = agg.manifest_blob_digest
		)
		SELECT DISTINCT advisory_id, relation, id AS version_id, artifact_id AS parent_id
		FROM Marked`,
		pgx.NamedArgs{"advisoryIds": advisoryIDs},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query marked artifact versions: %w", err)
	}
	artifactVersions, err := pgx.CollectRows(artifactRows, pgx.RowToStructByName[markedVersionRow])
	if err != nil {
		return nil, fmt.Errorf("could not get marked artifact versions: %w", err)
	}

	for _, row := range applicationVersions {
		marked := result[row.AdvisoryID]
		marked.AffectedApplicationVersions = append(
			marked.AffectedApplicationVersions,
			advisory.VersionRef{VersionID: row.VersionID, ParentID: row.ParentID},
		)
		result[row.AdvisoryID] = marked
	}
	for _, row := range artifactVersions {
		marked := result[row.AdvisoryID]
		ref := advisory.VersionRef{VersionID: row.VersionID, ParentID: row.ParentID}
		if row.Relation == types.AdvisoryVersionRelationFixed {
			marked.FixedArtifactVersions = append(marked.FixedArtifactVersions, ref)
		} else {
			marked.AffectedArtifactVersions = append(marked.AffectedArtifactVersions, ref)
		}
		result[row.AdvisoryID] = marked
	}
	return result, nil
}

// getPulledArtifactVersionIDs returns which of the given artifact versions the callers in
// scope have pulled from the registry.
//
// It takes the versions to look for rather than returning everything pulled, because
// ArtifactVersionPull is an append-only audit log that grows with every registry pull and is
// by far the largest table involved here. Narrowing to the versions the advisories actually
// mark drives the query off fk_ArtifactVersionPull_artifact_version_id and keeps it
// proportional to the advisories on screen instead of the organization's pull history. It
// costs nothing in accuracy: a pull of an unmarked version can never change the answer.
//
// The customer predicate cannot drive the query on its own, because coalescing the column
// with the membership fallback makes it unindexable.
//
// Attribution matches GetAdvisoryImpactedPulls, including that fallback through the pulling
// user's customer organization for pulls recorded before migration 71 added the column.
func getPulledArtifactVersionIDs(
	ctx context.Context, orgID uuid.UUID, scope AdvisoryScope, versionIDs []uuid.UUID,
) ([]uuid.UUID, error) {
	if len(versionIDs) == 0 || (scope.CustomerOrgID == nil && scope.PartnerOrgID == nil) {
		return nil, nil
	}
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT DISTINCT avpl.artifact_version_id
		FROM ArtifactVersionPull avpl
			JOIN ArtifactVersion av ON av.id = avpl.artifact_version_id
			JOIN Artifact a ON a.id = av.artifact_id
			LEFT JOIN Organization_UserAccount oua
				ON oua.user_account_id = avpl.useraccount_id
					AND oua.organization_id = a.organization_id
			LEFT JOIN CustomerOrganization co
				ON co.id = coalesce(avpl.customer_organization_id, oua.customer_organization_id)
		WHERE avpl.artifact_version_id = any(@versionIds)
			AND a.organization_id = @orgId
			AND (@partnerOrgId::uuid IS NULL OR co.partner_organization_id = @partnerOrgId)
			AND (@customerOrgId::uuid IS NULL
				OR coalesce(avpl.customer_organization_id, oua.customer_organization_id) = @customerOrgId)`,
		scope.bind(pgx.NamedArgs{"orgId": orgID, "versionIds": versionIDs}),
	)
	if err != nil {
		return nil, fmt.Errorf("could not query pulled artifact versions: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("could not get pulled artifact versions: %w", err)
	}
	return result, nil
}
