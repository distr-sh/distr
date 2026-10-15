package db

import (
	"context"
	"errors"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	overviewActivityLimit = 25
	overviewActivityDays  = 90
)

// overviewCustomerMemberJoin joins oua for the user u only when that user is a member of the customer
// organization, which decides whether a customer may see who did something.
const overviewCustomerMemberJoin = `
	LEFT JOIN Organization_UserAccount oua
		ON oua.user_account_id = u.id
			AND oua.organization_id = @orgId
			AND oua.customer_organization_id = @customerOrganizationId`

func GetOverviewDeployments(
	ctx context.Context,
	orgID, customerOrgID uuid.UUID,
) ([]types.OverviewDeployment, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT`+deploymentWithLatestRevisionOutputExpr+`,
			CASE WHEN oua.user_account_id IS NOT NULL THEN u.name END AS created_by_name,
			CASE WHEN oua.user_account_id IS NOT NULL THEN u.image_id END AS created_by_image_id
		FROM `+deploymentWithLatestRevisionFromExpr+`
			JOIN DeploymentTarget dt ON dt.id = d.deployment_target_id
			LEFT JOIN UserAccount u ON u.id = dr.created_by_user_account_id
			`+overviewCustomerMemberJoin+`
		WHERE dt.organization_id = @orgId AND dt.customer_organization_id = @customerOrganizationId
		ORDER BY d.created_at`,
		pgx.NamedArgs{"orgId": orgID, "customerOrganizationId": customerOrgID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query overview deployments: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.OverviewDeployment])
	if err != nil {
		return nil, fmt.Errorf("failed to collect overview deployments: %w", err)
	}
	for i := range result {
		if link, err := TemplateApplicationLink(
			result[i].ApplicationLinkTemplate, result[i].EnvFileData, result[i].ValuesYaml,
		); err == nil {
			result[i].ApplicationLink = link
		}
	}
	return result, nil
}

func GetOverviewChecklist(
	ctx context.Context,
	orgID, customerOrgID, userID uuid.UUID,
) (*types.OverviewChecklist, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`WITH `+customerOrgPullsExpr+`
		SELECT
			EXISTS (
				SELECT 1 FROM UserAccount u WHERE u.id = @userId AND (u.image_id IS NOT NULL OR u.mfa_enabled)
			) AS profile_completed,
			(
				SELECT count(*) FROM Organization_UserAccount oua
				WHERE oua.organization_id = @orgId AND oua.customer_organization_id = @customerOrganizationId
			) >= 2 AS team_members_invited,
			EXISTS (
				SELECT 1 FROM DeploymentTarget dt JOIN Deployment d ON d.deployment_target_id = dt.id
				WHERE dt.organization_id = @orgId AND dt.customer_organization_id = @customerOrganizationId
			) AS deployment_created,
			EXISTS (
				SELECT 1 FROM Secret s
				WHERE s.organization_id = @orgId AND s.customer_organization_id = @customerOrganizationId
			) AS secret_created,
			EXISTS (
				SELECT 1 FROM AccessToken t WHERE t.user_account_id = @userId AND t.organization_id = @orgId
			) AS access_token_created,
			EXISTS (
				SELECT 1 FROM customer_pull p
					JOIN ArtifactVersion v ON v.id = p.artifact_version_id
					JOIN Artifact a ON a.id = v.artifact_id
				WHERE a.organization_id = @orgId
			) AS artifact_pulled`,
		pgx.NamedArgs{"orgId": orgID, "customerOrganizationId": customerOrgID, "userId": userID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query overview checklist: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.OverviewChecklist])
	if err != nil {
		return nil, fmt.Errorf("failed to collect overview checklist: %w", err)
	}
	return &result, nil
}

// GetOverviewLicense returns the license key of the customer organization that decides its license state,
// preferring a valid key over one that is not valid yet over an expired one and among those the key that
// expires last. When it has no license key, it returns the entitlement that expires first, and nil when
// there is neither.
func GetOverviewLicense(ctx context.Context, orgID, customerOrgID uuid.UUID) (*types.OverviewLicense, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`WITH license_key AS (
			SELECT lk.name, lr.not_before, lr.expires_at
			FROM LicenseKey lk`+licenseKeyLatestRevisionJoin+`
			WHERE lk.organization_id = @orgId AND lk.customer_organization_id = @customerOrganizationId
				AND lr.payload IS NOT NULL
			ORDER BY
				CASE WHEN lr.expires_at <= now() THEN 2 WHEN lr.not_before > now() THEN 1 ELSE 0 END,
				lr.expires_at DESC,
				lk.name
			LIMIT 1
		),
		entitlement AS (
			SELECT name, expires_at
			FROM (
				SELECT name, expires_at FROM ApplicationEntitlement
				WHERE organization_id = @orgId AND customer_organization_id = @customerOrganizationId
				UNION ALL
				SELECT name, expires_at FROM ArtifactEntitlement
				WHERE organization_id = @orgId AND customer_organization_id = @customerOrganizationId
			) e
			ORDER BY expires_at NULLS LAST, name
			LIMIT 1
		)
		SELECT name, not_before, expires_at FROM license_key
		UNION ALL
		SELECT name, NULL, expires_at FROM entitlement WHERE NOT EXISTS (SELECT 1 FROM license_key)`,
		pgx.NamedArgs{"orgId": orgID, "customerOrganizationId": customerOrgID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query overview license: %w", err)
	}
	result, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[types.OverviewLicense])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to collect overview license: %w", err)
	}
	return result, nil
}

func GetOverviewAlertSummary(
	ctx context.Context,
	orgID, customerOrgID uuid.UUID,
) (*types.OverviewAlertSummary, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT
			(
				SELECT count(*) FROM AlertConfiguration c
				WHERE c.organization_id = @orgId AND c.customer_organization_id = @customerOrganizationId
			) AS alert_configuration_count,
			(
				SELECT count(*) FROM NotificationRecord r
				WHERE r.organization_id = @orgId
					AND r.customer_organization_id = @customerOrganizationId
					AND r.created_at > now() - interval '7 days'
					AND r.type IN ('alert', 'warning')
			) AS notification_count`,
		pgx.NamedArgs{"orgId": orgID, "customerOrganizationId": customerOrgID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query overview alert summary: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.OverviewAlertSummary])
	if err != nil {
		return nil, fmt.Errorf("failed to collect overview alert summary: %w", err)
	}
	return &result, nil
}

// GetOverviewSupportBundles returns the support bundles of the customer organization, most recently updated
// first. A bundle is updated by whoever last changed its status, or else by its creator.
func GetOverviewSupportBundles(
	ctx context.Context,
	orgID, customerOrgID uuid.UUID,
) ([]types.OverviewSupportBundle, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT sb.id, sb.title, sb.status, coalesce(sb.status_changed_at, sb.created_at) AS updated_at,
			CASE WHEN oua.user_account_id IS NOT NULL THEN u.name END AS user_name,
			CASE WHEN oua.user_account_id IS NOT NULL THEN u.image_id END AS user_image_id
		FROM SupportBundle sb
			LEFT JOIN UserAccount u ON u.id = CASE
				WHEN sb.status_changed_at IS NOT NULL THEN sb.status_changed_by_user_account_id
				ELSE sb.created_by_user_account_id
			END
			`+overviewCustomerMemberJoin+`
		WHERE sb.organization_id = @orgId AND sb.customer_organization_id = @customerOrganizationId
		ORDER BY updated_at DESC`,
		pgx.NamedArgs{"orgId": orgID, "customerOrganizationId": customerOrgID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query overview support bundles: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.OverviewSupportBundle])
	if err != nil {
		return nil, fmt.Errorf("failed to collect overview support bundles: %w", err)
	}
	return result, nil
}

// GetOverviewActivity returns the most recent events a customer organization is shown on its overview.
// Each branch is limited on its own, so that it can stop early on its created_at index, before the union
// is limited as a whole. A branch the filter excludes returns nothing.
//
// An artifact entitlement for specific versions covers every tag that points to the same manifest as one
// of them.
func GetOverviewActivity(
	ctx context.Context,
	orgID, customerOrgID uuid.UUID,
	filter types.OverviewActivityFilter,
) ([]types.OverviewActivity, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`WITH
		application_version_created AS (
			SELECT av.id, av.created_at, a.id AS resource_id, a.name AS subject, av.name AS detail
			FROM Application a
				JOIN ApplicationVersion av ON av.application_id = a.id
			WHERE @includeDeployments
				AND a.organization_id = @orgId
				AND av.created_at > now() - make_interval(days => @days::INT)
				AND av.archived_at IS NULL
				AND (NOT @entitledApplicationsOnly OR EXISTS (
					SELECT 1 FROM ApplicationEntitlement al
					WHERE al.application_id = a.id
						AND al.customer_organization_id = @customerOrganizationId
						AND (al.expires_at IS NULL OR al.expires_at > now())
						AND (
							NOT EXISTS (
								SELECT 1 FROM ApplicationEntitlement_ApplicationVersion alav
								WHERE alav.application_entitlement_id = al.id
							) OR EXISTS (
								SELECT 1 FROM ApplicationEntitlement_ApplicationVersion alav
								WHERE alav.application_entitlement_id = al.id AND alav.application_version_id = av.id
							)
						)
				))
			ORDER BY av.created_at DESC
			LIMIT @limit
		),
		artifact_version_created AS (
			SELECT v.id, v.created_at, ar.id AS resource_id, ar.name AS subject, v.name AS detail
			FROM Artifact ar
				JOIN ArtifactVersion v ON v.artifact_id = ar.id
			WHERE @includeArtifacts
				AND ar.organization_id = @orgId
				AND v.created_at > now() - make_interval(days => @days::INT)
				AND `+artifactVersionIsReleaseTagExpr("v")+`
				AND (NOT @entitledArtifactsOnly OR ar.public OR EXISTS (
					SELECT 1 FROM ArtifactEntitlement_Artifact ala
						JOIN ArtifactEntitlement al ON al.id = ala.artifact_entitlement_id
					WHERE ala.artifact_id = ar.id
						AND al.customer_organization_id = @customerOrganizationId
						AND (al.expires_at IS NULL OR al.expires_at > now())
						AND (ala.artifact_version_id IS NULL OR EXISTS (
							SELECT 1 FROM ArtifactVersion ev
							WHERE ev.id = ala.artifact_version_id AND ev.manifest_blob_digest = v.manifest_blob_digest
						))
				))
			ORDER BY v.created_at DESC
			LIMIT @limit
		),
		artifact_version_first_pulled AS (
			SELECT fp.id, fp.created_at, avc.resource_id, avc.subject, avc.detail,
				u.name AS user_name, u.image_id AS user_image_id
			FROM artifact_version_created avc
				CROSS JOIN LATERAL (
					SELECT DISTINCT ON (avpl.useraccount_id) avpl.id, avpl.created_at, avpl.useraccount_id
					FROM ArtifactVersionPull avpl
					WHERE avpl.artifact_version_id = avc.id
						AND avpl.useraccount_id IS NOT NULL
						AND `+artifactPullOfCustomerOrgExpr+`
					ORDER BY avpl.useraccount_id, avpl.created_at
				) fp
				JOIN UserAccount u ON u.id = fp.useraccount_id
		),
		deployment_revision_created AS (
			SELECT dr.id, dr.created_at, dt.id AS resource_id, a.name AS subject,
				av.name || ' on ' || dt.name AS detail,
				CASE WHEN oua.user_account_id IS NOT NULL THEN u.name END AS user_name,
				CASE WHEN oua.user_account_id IS NOT NULL THEN u.image_id END AS user_image_id
			FROM DeploymentTarget dt
				JOIN Deployment d ON d.deployment_target_id = dt.id
				JOIN DeploymentRevision dr ON dr.deployment_id = d.id
				JOIN ApplicationVersion av ON av.id = dr.application_version_id
				JOIN Application a ON a.id = av.application_id
				LEFT JOIN UserAccount u ON u.id = dr.created_by_user_account_id
				`+overviewCustomerMemberJoin+`
			WHERE @includeDeployments
				AND dt.organization_id = @orgId
				AND dt.customer_organization_id = @customerOrganizationId
				AND dr.created_at > now() - make_interval(days => @days::INT)
			ORDER BY dr.created_at DESC
			LIMIT @limit
		),
		license_key_revision AS (
			SELECT lr.id, lr.created_at, lk.id AS resource_id, lk.name AS subject,
				EXISTS (
					SELECT 1 FROM LicenseKeyRevision prev
					WHERE prev.license_key_id = lr.license_key_id AND prev.created_at < lr.created_at
				) AS is_update
			FROM LicenseKey lk
				JOIN LicenseKeyRevision lr ON lr.license_key_id = lk.id
			WHERE @includeLicensing
				AND lk.organization_id = @orgId
				AND lk.customer_organization_id = @customerOrganizationId
				AND lr.created_at > now() - make_interval(days => @days::INT)
			ORDER BY lr.created_at DESC
			LIMIT @limit
		),
		support_bundle_created AS (
			SELECT sb.id, sb.created_at, sb.id AS resource_id, sb.title AS subject,
				CASE WHEN oua.user_account_id IS NOT NULL THEN u.name END AS user_name,
				CASE WHEN oua.user_account_id IS NOT NULL THEN u.image_id END AS user_image_id
			FROM SupportBundle sb
				LEFT JOIN UserAccount u ON u.id = sb.created_by_user_account_id
				`+overviewCustomerMemberJoin+`
			WHERE @includeSupportBundles
				AND sb.organization_id = @orgId
				AND sb.customer_organization_id = @customerOrganizationId
				AND sb.created_at > now() - make_interval(days => @days::INT)
			ORDER BY sb.created_at DESC
			LIMIT @limit
		),
		support_bundle_status_changed AS (
			SELECT sb.id, sb.status_changed_at AS created_at, sb.id AS resource_id, sb.title AS subject, sb.status,
				CASE WHEN oua.user_account_id IS NOT NULL THEN u.name END AS user_name,
				CASE WHEN oua.user_account_id IS NOT NULL THEN u.image_id END AS user_image_id
			FROM SupportBundle sb
				LEFT JOIN UserAccount u ON u.id = sb.status_changed_by_user_account_id
				`+overviewCustomerMemberJoin+`
			WHERE @includeSupportBundles
				AND sb.organization_id = @orgId
				AND sb.customer_organization_id = @customerOrganizationId
				AND sb.status_changed_at > now() - make_interval(days => @days::INT)
			ORDER BY sb.status_changed_at DESC
			LIMIT @limit
		)
		SELECT * FROM (
			SELECT id, created_at, @typeApplicationVersionCreated::TEXT AS type, resource_id, subject, detail,
				NULL::TEXT AS user_name, NULL::UUID AS user_image_id
			FROM application_version_created
			UNION ALL
			SELECT id, created_at, @typeArtifactVersionCreated::TEXT, resource_id, subject, detail,
				NULL::TEXT, NULL::UUID
			FROM artifact_version_created
			UNION ALL
			SELECT id, created_at, @typeArtifactVersionFirstPulled::TEXT, resource_id, subject, detail,
				user_name, user_image_id
			FROM artifact_version_first_pulled
			UNION ALL
			SELECT id, created_at, @typeDeploymentRevisionCreated::TEXT, resource_id, subject, detail,
				user_name, user_image_id
			FROM deployment_revision_created
			UNION ALL
			SELECT id, created_at,
				CASE WHEN is_update THEN @typeLicenseKeyUpdated::TEXT ELSE @typeLicenseKeyCreated::TEXT END,
				resource_id, subject, NULL::TEXT AS detail, NULL::TEXT AS user_name, NULL::UUID AS user_image_id
			FROM license_key_revision
			UNION ALL
			SELECT id, created_at, @typeSupportBundleCreated::TEXT, resource_id, subject, NULL,
				user_name, user_image_id
			FROM support_bundle_created
			UNION ALL
			SELECT id, created_at,
				CASE status::TEXT
					WHEN @supportBundleStatusCreated::TEXT THEN @typeSupportBundleUploaded::TEXT
					WHEN @supportBundleStatusResolved::TEXT THEN @typeSupportBundleResolved::TEXT
					WHEN @supportBundleStatusCanceled::TEXT THEN @typeSupportBundleCanceled::TEXT
				END,
				resource_id, subject, NULL, user_name, user_image_id
			FROM support_bundle_status_changed
		) activity
		ORDER BY created_at DESC
		LIMIT @limit`,
		pgx.NamedArgs{
			"orgId":                    orgID,
			"customerOrganizationId":   customerOrgID,
			"includeDeployments":       filter.Deployments,
			"includeArtifacts":         filter.Artifacts,
			"includeLicensing":         filter.Licensing,
			"includeSupportBundles":    filter.SupportBundles,
			"entitledApplicationsOnly": filter.EntitledApplicationsOnly,
			"entitledArtifactsOnly":    filter.EntitledArtifactsOnly,
			"days":                     overviewActivityDays,
			"limit":                    overviewActivityLimit,

			"typeApplicationVersionCreated":  types.OverviewActivityTypeApplicationVersionCreated,
			"typeArtifactVersionCreated":     types.OverviewActivityTypeArtifactVersionCreated,
			"typeArtifactVersionFirstPulled": types.OverviewActivityTypeArtifactVersionFirstPulled,
			"typeDeploymentRevisionCreated":  types.OverviewActivityTypeDeploymentRevisionCreated,
			"typeLicenseKeyCreated":          types.OverviewActivityTypeLicenseKeyCreated,
			"typeLicenseKeyUpdated":          types.OverviewActivityTypeLicenseKeyUpdated,
			"typeSupportBundleCreated":       types.OverviewActivityTypeSupportBundleCreated,
			"typeSupportBundleUploaded":      types.OverviewActivityTypeSupportBundleUploaded,
			"typeSupportBundleResolved":      types.OverviewActivityTypeSupportBundleResolved,
			"typeSupportBundleCanceled":      types.OverviewActivityTypeSupportBundleCanceled,
			"supportBundleStatusCreated":     types.SupportBundleStatusCreated,
			"supportBundleStatusResolved":    types.SupportBundleStatusResolved,
			"supportBundleStatusCanceled":    types.SupportBundleStatusCanceled,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query overview activity: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.OverviewActivity])
	if err != nil {
		return nil, fmt.Errorf("failed to collect overview activity: %w", err)
	}
	return result, nil
}
