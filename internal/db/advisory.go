package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/advisory"
	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Advisories

const advisoryWithDetailsOutputExpr = `
	v.id,
	v.created_at,
	v.updated_at,
	v.organization_id,
	v.created_by_user_account_id,
	v.title,
	v.description,
	v.status,
	v.severity,
	v.cve_id,
	v.published_at,
	v.resolved_at,
	u.name AS created_by_user_name,
	u.image_id AS created_by_image_id,
	coalesce((
		SELECT array_agg(t.name ORDER BY t.name)
		FROM AdvisoryTag t
		WHERE t.advisory_id = v.id
	), ARRAY[]::TEXT[]) AS tags,
	(
		(SELECT count(*) FROM AdvisoryApplicationVersion vav
			WHERE vav.advisory_id = v.id AND vav.relation = 'affected')
		+ (SELECT count(*) FROM AdvisoryArtifactVersion vrv
			WHERE vrv.advisory_id = v.id AND vrv.relation = 'affected')
	) AS affected_version_count,
	(
		(SELECT count(*) FROM AdvisoryApplicationVersion vav
			WHERE vav.advisory_id = v.id AND vav.relation = 'fixed')
		+ (SELECT count(*) FROM AdvisoryArtifactVersion vrv
			WHERE vrv.advisory_id = v.id AND vrv.relation = 'fixed')
	) AS fixed_version_count,
	(SELECT count(*) FROM AdvisoryReference vr WHERE vr.advisory_id = v.id) AS reference_count
`

// filterCustomerVisible keeps only the advisories a customer organization may see.
// The rule itself lives in advisory.IsVisibleToCustomer; this only loads the data it
// needs. Filtering happens in Go rather than SQL because the rule is security critical and
// needs to be unit testable, which is affordable here because the result set is a bounded
// per-organization list with no pagination.
func filterCustomerVisible(
	ctx context.Context,
	advisories []types.AdvisoryWithDetails,
	orgID, customerOrgID uuid.UUID,
) ([]types.AdvisoryWithDetails, error) {
	if len(advisories) == 0 {
		return advisories, nil
	}

	view, err := GetCustomerView(ctx, orgID, customerOrgID)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, len(advisories))
	for i, v := range advisories {
		ids[i] = v.ID
	}
	affected, err := GetAffectedVersions(ctx, ids)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	visible := make([]types.AdvisoryWithDetails, 0, len(advisories))
	for _, v := range advisories {
		versions := affected[v.ID]
		if advisory.IsVisibleToCustomer(
			v.Status, versions.ApplicationVersions, versions.ArtifactVersions, view, now,
		) {
			visible = append(visible, v)
		}
	}
	return visible, nil
}

type AdvisoryFilter struct {
	// CustomerOrgID scopes the result to what a customer may see. Nil for vendor users.
	CustomerOrgID *uuid.UUID
	// Each of the following matches an advisory that has any of the given values.
	// An empty slice means the filter is not applied at all.
	Statuses   []types.AdvisoryStatus
	Severities []types.AdvisorySeverity
	Tags       []string
}

// stringsOf converts a slice of a string-based enum type into plain strings, so that it can be
// passed as a text[] query parameter.
func stringsOf[T ~string](values []T) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = string(value)
	}
	return result
}

func GetAdvisories(
	ctx context.Context, orgID uuid.UUID, filter AdvisoryFilter,
) ([]types.AdvisoryWithDetails, error) {
	db := internalctx.GetDb(ctx)

	args := pgx.NamedArgs{"orgId": orgID}

	query := fmt.Sprintf(`
		SELECT %v
		FROM Advisory v
			LEFT JOIN UserAccount u ON v.created_by_user_account_id = u.id
		WHERE v.organization_id = @orgId`,
		advisoryWithDetailsOutputExpr)

	// The enum columns are compared as text so that the parameter is a plain text[], which
	// avoids having to teach pgx how to encode an array of a custom enum type.
	if len(filter.Statuses) > 0 {
		args["statuses"] = stringsOf(filter.Statuses)
		query += ` AND v.status::text = any(@statuses)`
	}
	if len(filter.Severities) > 0 {
		args["severities"] = stringsOf(filter.Severities)
		query += ` AND v.severity::text = any(@severities)`
	}
	if len(filter.Tags) > 0 {
		args["tags"] = filter.Tags
		query += ` AND EXISTS (
			SELECT 1 FROM AdvisoryTag t
			WHERE t.advisory_id = v.id AND t.name = any(@tags)
		)`
	}

	query += ` ORDER BY v.created_at DESC`

	rows, err := db.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("could not query advisories: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.AdvisoryWithDetails])
	if err != nil {
		return nil, fmt.Errorf("could not get advisories: %w", err)
	}

	if filter.CustomerOrgID != nil {
		return filterCustomerVisible(ctx, result, orgID, *filter.CustomerOrgID)
	}
	return result, nil
}

func GetAdvisoryByID(
	ctx context.Context, id, orgID uuid.UUID, customerOrgID *uuid.UUID,
) (*types.AdvisoryWithDetails, error) {
	db := internalctx.GetDb(ctx)

	args := pgx.NamedArgs{"id": id, "orgId": orgID}
	query := fmt.Sprintf(`
		SELECT %v
		FROM Advisory v
			LEFT JOIN UserAccount u ON v.created_by_user_account_id = u.id
		WHERE v.id = @id AND v.organization_id = @orgId`,
		advisoryWithDetailsOutputExpr)

	rows, err := db.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("could not query advisory: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.AdvisoryWithDetails])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not get advisory: %w", err)
	}

	if customerOrgID != nil {
		// An advisory the customer may not see must be indistinguishable from one that
		// does not exist, so this reports ErrNotFound rather than a permission error.
		visible, err := filterCustomerVisible(
			ctx, []types.AdvisoryWithDetails{result}, orgID, *customerOrgID,
		)
		if err != nil {
			return nil, err
		}
		if len(visible) == 0 {
			return nil, apierrors.ErrNotFound
		}
	}
	return &result, nil
}

// CreateAdvisory inserts the advisory. Status must be set by the caller; the column
// default is not relied upon, because the initial status depends on how the advisory was
// reported.
func CreateAdvisory(ctx context.Context, advisory *types.Advisory) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`INSERT INTO Advisory
			(organization_id, created_by_user_account_id, title, description, status, severity, cve_id)
		VALUES (@orgId, @userId, @title, @description, @status, @severity, @cveId)
		RETURNING id, created_at, updated_at, organization_id, created_by_user_account_id,
			title, description, status, severity, cve_id, published_at, resolved_at`,
		pgx.NamedArgs{
			"orgId":       advisory.OrganizationID,
			"userId":      advisory.CreatedByUserAccountID,
			"title":       advisory.Title,
			"description": advisory.Description,
			"status":      advisory.Status,
			"severity":    advisory.Severity,
			"cveId":       advisory.CveID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not create advisory: %w", errDuplicateCveID(err))
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.Advisory])
	if err != nil {
		return fmt.Errorf("could not create advisory: %w", errDuplicateCveID(err))
	}
	*advisory = result
	return nil
}

// errDuplicateCveID translates a unique violation into a conflict, so that handlers can
// report the duplicate rather than a server error. Advisory has exactly one unique index
// besides its generated primary key, so a violation here is always the CVE ID.
func errDuplicateCveID(err error) error {
	if pgerr, ok := errors.AsType[*pgconn.PgError](err); ok && pgerr.Code == pgerrcode.UniqueViolation {
		return fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
	}
	return err
}

func UpdateAdvisory(ctx context.Context, advisory *types.Advisory) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`UPDATE Advisory
		SET title = @title,
			description = @description,
			severity = @severity,
			cve_id = @cveId,
			updated_at = now()
		WHERE id = @id AND organization_id = @orgId
		RETURNING id, created_at, updated_at, organization_id, created_by_user_account_id,
			title, description, status, severity, cve_id, published_at, resolved_at`,
		pgx.NamedArgs{
			"id":          advisory.ID,
			"orgId":       advisory.OrganizationID,
			"title":       advisory.Title,
			"description": advisory.Description,
			"severity":    advisory.Severity,
			"cveId":       advisory.CveID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not update advisory: %w", errDuplicateCveID(err))
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.Advisory])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apierrors.ErrNotFound
		}
		return fmt.Errorf("could not update advisory: %w", errDuplicateCveID(err))
	}
	*advisory = result
	return nil
}

// UpdateAdvisoryStatus applies a status change and maintains the published_at and
// resolved_at timestamps. published_at is only set the first time an advisory is
// published so that unpublishing and republishing keeps the original disclosure date.
func UpdateAdvisoryStatus(
	ctx context.Context, id, orgID uuid.UUID, from, status types.AdvisoryStatus,
) error {
	db := internalctx.GetDb(ctx)
	result, err := db.Exec(
		ctx,
		// Every occurrence of the status parameter carries the same explicit cast. NamedArgs
		// collapses them into one placeholder, and without the cast Postgres deduces the enum
		// from the assignment but text from the IN list and rejects the statement (42P08).
		//
		// The WHERE clause pins the update to the status the caller validated the transition
		// against. If a concurrent request already moved the advisory to a different status,
		// this update matches no rows so the stale transition cannot be applied.
		`UPDATE Advisory
		SET status = @status::advisory_status,
			published_at = CASE
				WHEN @status::advisory_status IN ('published', 'resolved') AND published_at IS NULL
					THEN now()
				ELSE published_at
			END,
			resolved_at = CASE
				WHEN @status::advisory_status = 'resolved' THEN now()
				ELSE NULL
			END,
			updated_at = now()
		WHERE id = @id AND organization_id = @orgId AND status = @from::advisory_status`,
		pgx.NamedArgs{"id": id, "orgId": orgID, "from": from, "status": status},
	)
	if err != nil {
		return fmt.Errorf("could not update advisory status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return apierrors.ErrConflict
	}
	return nil
}

// Tags

func GetAdvisoryTagNames(ctx context.Context, orgID uuid.UUID) ([]string, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT DISTINCT t.name
		FROM AdvisoryTag t
			JOIN Advisory v ON v.id = t.advisory_id
		WHERE v.organization_id = @orgId
		ORDER BY t.name`,
		pgx.NamedArgs{"orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query advisory tags: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("could not get advisory tags: %w", err)
	}
	return result, nil
}

func SetAdvisoryTags(ctx context.Context, advisoryID uuid.UUID, tags []string) error {
	db := internalctx.GetDb(ctx)

	if _, err := db.Exec(
		ctx,
		`DELETE FROM AdvisoryTag WHERE advisory_id = @id`,
		pgx.NamedArgs{"id": advisoryID},
	); err != nil {
		return fmt.Errorf("could not delete existing advisory tags: %w", err)
	}

	if len(tags) > 0 {
		if _, err := db.CopyFrom(
			ctx,
			pgx.Identifier{"advisorytag"},
			[]string{"advisory_id", "name"},
			pgx.CopyFromSlice(len(tags), func(i int) ([]any, error) {
				return []any{advisoryID, tags[i]}, nil
			}),
		); err != nil {
			return fmt.Errorf("could not insert advisory tags: %w", err)
		}
	}

	return nil
}

// References

func GetAdvisoryReferences(
	ctx context.Context, advisoryID uuid.UUID,
) ([]types.AdvisoryReference, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT id, advisory_id, url, label
		FROM AdvisoryReference
		WHERE advisory_id = @id
		ORDER BY url`,
		pgx.NamedArgs{"id": advisoryID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query advisory references: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.AdvisoryReference])
	if err != nil {
		return nil, fmt.Errorf("could not get advisory references: %w", err)
	}
	return result, nil
}

func SetAdvisoryReferences(
	ctx context.Context, advisoryID uuid.UUID, references []types.AdvisoryReference,
) error {
	db := internalctx.GetDb(ctx)

	if _, err := db.Exec(
		ctx,
		`DELETE FROM AdvisoryReference WHERE advisory_id = @id`,
		pgx.NamedArgs{"id": advisoryID},
	); err != nil {
		return fmt.Errorf("could not delete existing advisory references: %w", err)
	}

	if len(references) > 0 {
		if _, err := db.CopyFrom(
			ctx,
			pgx.Identifier{"advisoryreference"},
			[]string{"advisory_id", "url", "label"},
			pgx.CopyFromSlice(len(references), func(i int) ([]any, error) {
				return []any{advisoryID, references[i].URL, references[i].Label}, nil
			}),
		); err != nil {
			return fmt.Errorf("could not insert advisory references: %w", err)
		}
	}

	return nil
}

// Versions

func GetAdvisoryApplicationVersions(
	ctx context.Context, advisoryID uuid.UUID,
) ([]types.AdvisoryApplicationVersion, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT vav.advisory_id, vav.relation,
			a.id AS application_id, a.name AS application_name,
			av.id AS application_version_id, av.name AS application_version_name
		FROM AdvisoryApplicationVersion vav
			JOIN ApplicationVersion av ON av.id = vav.application_version_id
			JOIN Application a ON a.id = av.application_id
		WHERE vav.advisory_id = @id
		ORDER BY a.name, av.created_at`,
		pgx.NamedArgs{"id": advisoryID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query advisory application versions: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.AdvisoryApplicationVersion])
	if err != nil {
		return nil, fmt.Errorf("could not get advisory application versions: %w", err)
	}
	return result, nil
}

func GetAdvisoryArtifactVersions(
	ctx context.Context, advisoryID uuid.UUID,
) ([]types.AdvisoryArtifactVersion, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT vrv.advisory_id, vrv.relation,
			a.id AS artifact_id, a.name AS artifact_name,
			av.id AS artifact_version_id, av.name AS artifact_version_name
		FROM AdvisoryArtifactVersion vrv
			JOIN ArtifactVersion av ON av.id = vrv.artifact_version_id
			JOIN Artifact a ON a.id = av.artifact_id
		WHERE vrv.advisory_id = @id
		ORDER BY a.name, av.created_at`,
		pgx.NamedArgs{"id": advisoryID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query advisory artifact versions: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.AdvisoryArtifactVersion])
	if err != nil {
		return nil, fmt.Errorf("could not get advisory artifact versions: %w", err)
	}
	return result, nil
}

// AdvisoryVersionSelection is the set of versions an advisory applies to,
// as submitted by the client.
type AdvisoryVersionSelection struct {
	AffectedApplicationVersionIDs []uuid.UUID
	FixedApplicationVersionIDs    []uuid.UUID
	AffectedArtifactVersionIDs    []uuid.UUID
	FixedArtifactVersionIDs       []uuid.UUID
}

type versionRelationRow struct {
	versionID uuid.UUID
	relation  types.AdvisoryVersionRelation
}

func relationRows(
	affected, fixed []uuid.UUID,
) []versionRelationRow {
	rows := make([]versionRelationRow, 0, len(affected)+len(fixed))
	for _, id := range affected {
		rows = append(rows, versionRelationRow{id, types.AdvisoryVersionRelationAffected})
	}
	for _, id := range fixed {
		rows = append(rows, versionRelationRow{id, types.AdvisoryVersionRelationFixed})
	}
	return rows
}

func SetAdvisoryVersions(
	ctx context.Context, advisoryID uuid.UUID, selection AdvisoryVersionSelection,
) error {
	db := internalctx.GetDb(ctx)

	if _, err := db.Exec(
		ctx,
		`DELETE FROM AdvisoryApplicationVersion WHERE advisory_id = @id`,
		pgx.NamedArgs{"id": advisoryID},
	); err != nil {
		return fmt.Errorf("could not delete existing advisory application versions: %w", err)
	}
	if _, err := db.Exec(
		ctx,
		`DELETE FROM AdvisoryArtifactVersion WHERE advisory_id = @id`,
		pgx.NamedArgs{"id": advisoryID},
	); err != nil {
		return fmt.Errorf("could not delete existing advisory artifact versions: %w", err)
	}

	appRows := relationRows(selection.AffectedApplicationVersionIDs, selection.FixedApplicationVersionIDs)
	if len(appRows) > 0 {
		if _, err := db.CopyFrom(
			ctx,
			pgx.Identifier{"advisoryapplicationversion"},
			[]string{"advisory_id", "application_version_id", "relation"},
			pgx.CopyFromSlice(len(appRows), func(i int) ([]any, error) {
				return []any{advisoryID, appRows[i].versionID, appRows[i].relation}, nil
			}),
		); err != nil {
			return fmt.Errorf("could not insert advisory application versions: %w", err)
		}
	}

	artifactRows := relationRows(selection.AffectedArtifactVersionIDs, selection.FixedArtifactVersionIDs)
	if len(artifactRows) > 0 {
		if _, err := db.CopyFrom(
			ctx,
			pgx.Identifier{"advisoryartifactversion"},
			[]string{"advisory_id", "artifact_version_id", "relation"},
			pgx.CopyFromSlice(len(artifactRows), func(i int) ([]any, error) {
				return []any{advisoryID, artifactRows[i].versionID, artifactRows[i].relation}, nil
			}),
		); err != nil {
			return fmt.Errorf("could not insert advisory artifact versions: %w", err)
		}
	}

	return nil
}

// CountAdvisoryVersionsOutsideOrg reports how many of the given version IDs do not
// belong to the organization. Callers use this to reject a selection that would attach
// another tenant's versions to an advisory.
func CountAdvisoryVersionsOutsideOrg(
	ctx context.Context, orgID uuid.UUID, applicationVersionIDs, artifactVersionIDs []uuid.UUID,
) (int64, error) {
	db := internalctx.GetDb(ctx)
	var count int64
	err := db.QueryRow(
		ctx,
		`SELECT
			(SELECT count(*)
				FROM unnest(@applicationVersionIds::UUID[]) AS ids(id)
				WHERE NOT EXISTS (
					SELECT 1 FROM ApplicationVersion av
						JOIN Application a ON a.id = av.application_id
					WHERE av.id = ids.id AND a.organization_id = @orgId
				))
			+ (SELECT count(*)
				FROM unnest(@artifactVersionIds::UUID[]) AS ids(id)
				WHERE NOT EXISTS (
					SELECT 1 FROM ArtifactVersion av
						JOIN Artifact a ON a.id = av.artifact_id
					WHERE av.id = ids.id AND a.organization_id = @orgId
				))`,
		pgx.NamedArgs{
			"orgId":                 orgID,
			"applicationVersionIds": applicationVersionIDs,
			"artifactVersionIds":    artifactVersionIDs,
		},
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("could not validate advisory versions: %w", err)
	}
	return count, nil
}

// Events

func GetAdvisoryEvents(
	ctx context.Context, advisoryID uuid.UUID,
) ([]types.AdvisoryEventWithUser, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT e.id, e.created_at, e.advisory_id, e.user_account_id, e.type, e.message,
			u.name AS user_name, u.image_id AS user_image_id
		FROM AdvisoryEvent e
			LEFT JOIN UserAccount u ON e.user_account_id = u.id
		WHERE e.advisory_id = @id
		ORDER BY e.created_at`,
		pgx.NamedArgs{"id": advisoryID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query advisory events: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.AdvisoryEventWithUser])
	if err != nil {
		return nil, fmt.Errorf("could not get advisory events: %w", err)
	}
	return result, nil
}

func CreateAdvisoryEvent(
	ctx context.Context,
	advisoryID uuid.UUID,
	userID *uuid.UUID,
	eventType types.AdvisoryEventType,
	message *string,
) error {
	db := internalctx.GetDb(ctx)
	if _, err := db.Exec(
		ctx,
		`INSERT INTO AdvisoryEvent (advisory_id, user_account_id, type, message)
		VALUES (@advisoryId, @userId, @type, @message)`,
		pgx.NamedArgs{
			"advisoryId": advisoryID,
			"userId":     userID,
			"type":       eventType,
			"message":    message,
		},
	); err != nil {
		return fmt.Errorf("could not create advisory event: %w", err)
	}
	return nil
}

func CreateAdvisoryCommentEvent(
	ctx context.Context, advisoryID uuid.UUID, userID uuid.UUID, content string,
) (*types.AdvisoryEventWithUser, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`WITH inserted AS (
			INSERT INTO AdvisoryEvent (advisory_id, user_account_id, type, message)
			VALUES (@advisoryId, @userId, 'comment', @message)
			RETURNING *
		)
		SELECT i.id, i.created_at, i.advisory_id, i.user_account_id, i.type, i.message,
			u.name AS user_name, u.image_id AS user_image_id
		FROM inserted i
			LEFT JOIN UserAccount u ON i.user_account_id = u.id`,
		pgx.NamedArgs{
			"advisoryId": advisoryID,
			"userId":     userID,
			"message":    content,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not create advisory comment: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.AdvisoryEventWithUser])
	if err != nil {
		return nil, fmt.Errorf("could not create advisory comment: %w", err)
	}
	return &result, nil
}

// Impact

// GetAdvisoryImpactedDeployments returns one row per deployment that has ever run an
// application version this advisory affects, classified by the version its current
// revision runs: still affected, patched, or moved onto a version marked neither.
//
// Each row also carries the most recent affected version the deployment ran and when, so
// that the exposure window stays visible for deployments that have since moved on.
func GetAdvisoryImpactedDeployments(
	ctx context.Context, advisoryID, orgID uuid.UUID, partnerOrgID *uuid.UUID,
) ([]types.AdvisoryImpactedDeployment, error) {
	db := internalctx.GetDb(ctx)
	isVendor := partnerOrgID == nil
	rows, err := db.Query(
		ctx,
		// marked holds both relations, because the state of a deployment depends on whether the
		// version it runs now is affected, fixed, or neither.
		`WITH marked AS (
			SELECT vav.application_version_id, vav.relation
			FROM AdvisoryApplicationVersion vav
				JOIN Advisory v ON v.id = vav.advisory_id
			WHERE vav.advisory_id = @id
				AND v.organization_id = @orgId
		), impacted AS (
			SELECT DISTINCT ON (dr.deployment_id)
				dr.deployment_id,
				dr.application_version_id,
				dr.created_at AS last_deployed_at
			FROM DeploymentRevision dr
				JOIN marked ON marked.application_version_id = dr.application_version_id
					AND marked.relation = 'affected'
			ORDER BY dr.deployment_id, dr.created_at DESC
		), current_revision AS (
			SELECT DISTINCT ON (dr.deployment_id) dr.deployment_id, dr.application_version_id
			FROM DeploymentRevision dr
			WHERE dr.deployment_id IN (SELECT deployment_id FROM impacted)
			ORDER BY dr.deployment_id, dr.created_at DESC
		), classified AS (
			SELECT
				dt.customer_organization_id,
				co.name AS customer_organization_name,
				d.id AS deployment_id,
				dt.id AS deployment_target_id,
				dt.name AS deployment_target_name,
				a.id AS application_id,
				a.name AS application_name,
				av.id AS application_version_id,
				av.name AS application_version_name,
				current_av.id AS current_application_version_id,
				current_av.name AS current_application_version_name,
				CASE
					WHEN cr.application_version_id IN (
						SELECT application_version_id FROM marked WHERE relation = 'affected')
						THEN 'affected'
					WHEN cr.application_version_id IN (
						SELECT application_version_id FROM marked WHERE relation = 'fixed')
						THEN 'patched'
					ELSE 'not_affected'
				END AS state,
				i.last_deployed_at
			FROM impacted i
				JOIN Deployment d ON d.id = i.deployment_id
				JOIN DeploymentTarget dt ON dt.id = d.deployment_target_id
				JOIN ApplicationVersion av ON av.id = i.application_version_id
				JOIN Application a ON a.id = av.application_id
				JOIN current_revision cr ON cr.deployment_id = i.deployment_id
				JOIN ApplicationVersion current_av ON current_av.id = cr.application_version_id
				LEFT JOIN CustomerOrganization co ON co.id = dt.customer_organization_id
			WHERE dt.organization_id = @orgId
				AND (@isVendor OR co.partner_organization_id = @partnerOrgId)
		)
		SELECT * FROM classified
		ORDER BY
			CASE state WHEN 'affected' THEN 0 WHEN 'not_affected' THEN 1 ELSE 2 END,
			customer_organization_name NULLS LAST,
			deployment_target_name`,
		pgx.NamedArgs{
			"id":           advisoryID,
			"orgId":        orgID,
			"isVendor":     isVendor,
			"partnerOrgId": partnerOrgID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query advisory impacted deployments: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.AdvisoryImpactedDeployment])
	if err != nil {
		return nil, fmt.Errorf("could not get advisory impacted deployments: %w", err)
	}
	return result, nil
}

// GetAdvisoryImpactedPulls aggregates registry pulls of the artifact versions this
// advisory affects.
//
// Two subtleties are handled here. A pull is recorded against whichever ArtifactVersion row
// the client referenced, tag or digest, so the selected version is expanded to every row
// sharing its manifest digest; otherwise selecting a digest would miss all tag-based pulls.
// And ArtifactVersionPull.customer_organization_id only exists since migration 71 and was
// never backfilled, so older pulls are attributed through the pulling user's customer
// organization membership instead.
func GetAdvisoryImpactedPulls(
	ctx context.Context, advisoryID, orgID uuid.UUID, partnerOrgID *uuid.UUID,
) ([]types.AdvisoryImpactedPull, error) {
	db := internalctx.GetDb(ctx)
	isVendor := partnerOrgID == nil
	rows, err := db.Query(
		ctx,
		`WITH affected AS (
			SELECT DISTINCT sibling.id AS artifact_version_id
			FROM AdvisoryArtifactVersion vrv
				JOIN Advisory v ON v.id = vrv.advisory_id
				JOIN ArtifactVersion selected ON selected.id = vrv.artifact_version_id
				JOIN ArtifactVersion sibling
					ON sibling.artifact_id = selected.artifact_id
					AND sibling.manifest_blob_digest = selected.manifest_blob_digest
			WHERE vrv.advisory_id = @id
				AND vrv.relation = 'affected'
				AND v.organization_id = @orgId
		)
		SELECT
			coalesce(avpl.customer_organization_id, oua.customer_organization_id)
				AS customer_organization_id,
			co.name AS customer_organization_name,
			a.id AS artifact_id,
			a.name AS artifact_name,
			av.id AS artifact_version_id,
			av.name AS artifact_version_name,
			count(*) AS pull_count,
			max(avpl.created_at) AS last_pulled_at
		FROM ArtifactVersionPull avpl
			JOIN affected ON affected.artifact_version_id = avpl.artifact_version_id
			JOIN ArtifactVersion av ON av.id = avpl.artifact_version_id
			JOIN Artifact a ON a.id = av.artifact_id
			LEFT JOIN Organization_UserAccount oua
				ON oua.user_account_id = avpl.useraccount_id
					AND oua.organization_id = a.organization_id
			LEFT JOIN CustomerOrganization co
				ON co.id = coalesce(avpl.customer_organization_id, oua.customer_organization_id)
		WHERE a.organization_id = @orgId
			AND (@isVendor OR co.partner_organization_id = @partnerOrgId)
		GROUP BY coalesce(avpl.customer_organization_id, oua.customer_organization_id),
			co.name, a.id, a.name, av.id, av.name
		ORDER BY co.name NULLS LAST, a.name, av.name`,
		pgx.NamedArgs{
			"id":           advisoryID,
			"orgId":        orgID,
			"isVendor":     isVendor,
			"partnerOrgId": partnerOrgID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query advisory impacted pulls: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.AdvisoryImpactedPull])
	if err != nil {
		return nil, fmt.Errorf("could not get advisory impacted pulls: %w", err)
	}
	return result, nil
}
