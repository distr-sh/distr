package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	applicationOutputExpr        = `a.id, a.created_at, a.organization_id, a.name, a.type, a.image_id`
	applicationVersionOutputExpr = `av.id, av.created_at, av.archived_at, av.name, av.link_template, av.application_id,
		av.chart_type, av.chart_name, av.chart_url, av.chart_version, av.values_file_data, av.template_file_data,
	 av.compose_file_data`
	applicationWithVersionsOutputExpr = applicationOutputExpr + `,
		coalesce((
			SELECT array_agg(row(av.id, av.created_at, av.archived_at, av.name, av.link_template, av.application_id,
				av.chart_type, av.chart_name, av.chart_url, av.chart_version) ORDER BY av.created_at ASC)
			FROM ApplicationVersion av
			WHERE av.application_id = a.id
		), array[]::record[]) AS versions `

	applicationWithEntitledVersionsOutputExpr = applicationOutputExpr + `,
		coalesce((
			SELECT array_agg(row(av.id, av.created_at, av.archived_at, av.name, av.link_template, av.application_id,
				av.chart_type, av.chart_name, av.chart_url, av.chart_version) ORDER BY av.created_at ASC)
			FROM ApplicationVersion av
			WHERE av.application_id = a.id and
				((av.id IN
					(SELECT application_version_id FROM ApplicationEntitlement_ApplicationVersion
						WHERE application_entitlement_id = al.id)
					OR (SELECT NOT EXISTS (SELECT FROM ApplicationEntitlement_ApplicationVersion
						WHERE application_entitlement_id = al.id)))
		)), array[]::record[]) AS versions `
)

func CreateApplication(ctx context.Context, application *types.Application, orgID uuid.UUID) error {
	application.OrganizationID = orgID
	db := internalctx.GetDb(ctx)
	row := db.QueryRow(ctx,
		"INSERT INTO Application (name, type, organization_id) VALUES (@name, @type, @orgId) RETURNING id, created_at",
		pgx.NamedArgs{"name": application.Name, "type": application.Type, "orgId": application.OrganizationID})
	if err := row.Scan(&application.ID, &application.CreatedAt); err != nil {
		return fmt.Errorf("could not save application: %w", err)
	}
	return nil
}

func UpdateApplication(ctx context.Context, application *types.Application, orgID uuid.UUID) error {
	application.OrganizationID = orgID
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"UPDATE Application SET name = @name WHERE id = @id AND organization_id = @orgId RETURNING *",
		pgx.NamedArgs{"id": application.ID, "name": application.Name, "orgId": application.OrganizationID})
	if err != nil {
		return fmt.Errorf("could not update application: %w", err)
	} else if updated, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByNameLax[types.Application]); err != nil {
		return fmt.Errorf("could not get updated application: %w", err)
	} else {
		*application = updated
		return nil
	}
}

func DeleteApplicationWithID(ctx context.Context, id uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx, `DELETE FROM Application WHERE id = @id`, pgx.NamedArgs{"id": id})
	if err != nil {
		if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.ForeignKeyViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
	} else if cmd.RowsAffected() == 0 {
		err = apierrors.ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("could not delete Application: %w", err)
	}

	return nil
}

func GetApplicationsByOrgID(ctx context.Context, orgID uuid.UUID) ([]types.Application, error) {
	db := internalctx.GetDb(ctx)
	if rows, err := db.Query(ctx, `
			SELECT `+applicationWithVersionsOutputExpr+`
			FROM Application a
			WHERE a.organization_id = @orgId
			ORDER BY a.name
			`, pgx.NamedArgs{"orgId": orgID}); err != nil {
		return nil, fmt.Errorf("failed to query applications: %w", err)
	} else if applications, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Application]); err != nil {
		return nil, fmt.Errorf("failed to get applications: %w", err)
	} else {
		return applications, nil
	}
}

func appendMissingVersions(application types.Application, versions []types.ApplicationVersion) types.Application {
	for _, version := range versions {
		found := false
		for _, existingVersion := range application.Versions {
			if version.ID == existingVersion.ID {
				found = true
				break
			}
		}
		if !found {
			application.Versions = append(application.Versions, version)
		}
	}
	return application
}

func mergeApplications(applications []types.Application) []types.Application {
	applicationMap := make(map[uuid.UUID]types.Application)
	for _, application := range applications {
		if existingApplication, ok := applicationMap[application.ID]; ok {
			applicationMap[application.ID] = appendMissingVersions(existingApplication, application.Versions)
		} else {
			applicationMap[application.ID] = application
		}
	}
	return util.GetValues(applicationMap)
}

func GetApplicationsWithEntitlementOwnerID(
	ctx context.Context,
	customerOrganizationID uuid.UUID,
) ([]types.Application, error) {
	db := internalctx.GetDb(ctx)
	if rows, err := db.Query(ctx, `
			SELECT DISTINCT `+applicationWithEntitledVersionsOutputExpr+`
			FROM ApplicationEntitlement al
				LEFT JOIN Application a ON al.application_id = a.id
			WHERE al.customer_organization_id = @id AND (al.expires_at IS NULL OR al.expires_at > now())
			ORDER BY a.name
			`, pgx.NamedArgs{"id": customerOrganizationID}); err != nil {
		return nil, fmt.Errorf("failed to query applications: %w", err)
	} else if applications, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Application]); err != nil {
		return nil, fmt.Errorf("failed to get applications: %w", err)
	} else {
		return mergeApplications(applications), nil
	}
}

func GetApplication(ctx context.Context, id, orgID uuid.UUID) (*types.Application, error) {
	db := internalctx.GetDb(ctx)
	if rows, err := db.Query(ctx, `
			SELECT `+applicationWithVersionsOutputExpr+`
			FROM Application a
			WHERE a.id = @id AND a.organization_id = @orgId
		`, pgx.NamedArgs{"id": id, "orgId": orgID}); err != nil {
		return nil, fmt.Errorf("failed to query application: %w", err)
	} else if application, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.Application]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get application: %w", err)
	} else {
		return &application, nil
	}
}

func GetApplicationWithEntitlementOwnerID(
	ctx context.Context,
	customerOrganizationID uuid.UUID,
	id uuid.UUID,
) (*types.Application, error) {
	db := internalctx.GetDb(ctx)
	if rows, err := db.Query(ctx, `
			SELECT DISTINCT `+applicationWithEntitledVersionsOutputExpr+`
			FROM ApplicationEntitlement al
				LEFT JOIN Application a ON al.application_id = a.id
			WHERE al.customer_organization_id = @ownerID AND a.id = @id AND (al.expires_at IS NULL OR al.expires_at > now())
			ORDER BY a.name
			`, pgx.NamedArgs{"ownerID": customerOrganizationID, "id": id}); err != nil {
		return nil, fmt.Errorf("failed to query applications: %w", err)
	} else if applications, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Application]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get application: %w", err)
	} else {
		return &mergeApplications(applications)[0], nil
	}
}

func GetApplicationForApplicationVersionID(ctx context.Context, id, orgID uuid.UUID) (*types.Application, error) {
	db := internalctx.GetDb(ctx)
	if rows, err := db.Query(ctx, `
			SELECT `+applicationWithVersionsOutputExpr+`
			FROM ApplicationVersion v
				LEFT JOIN Application a ON a.id = v.application_id
			WHERE v.id = @id AND a.organization_id = @orgId
		`, pgx.NamedArgs{"id": id, "orgId": orgID}); err != nil {
		return nil, fmt.Errorf("failed to query application: %w", err)
	} else if application, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.Application]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get application: %w", err)
	} else {
		return &application, nil
	}
}

func CreateApplicationVersion(ctx context.Context, applicationVersion *types.ApplicationVersion) error {
	db := internalctx.GetDb(ctx)

	args := pgx.NamedArgs{
		"name":          applicationVersion.Name,
		"linkTemplate":  applicationVersion.LinkTemplate,
		"applicationId": applicationVersion.ApplicationID,
		"chartType":     applicationVersion.ChartType,
		"chartName":     applicationVersion.ChartName,
		"chartUrl":      applicationVersion.ChartUrl,
		"chartVersion":  applicationVersion.ChartVersion,
	}
	if applicationVersion.ComposeFileData != nil {
		args["composeFileData"] = applicationVersion.ComposeFileData
	}
	if applicationVersion.ValuesFileData != nil {
		args["valuesFileData"] = applicationVersion.ValuesFileData
	}
	if applicationVersion.TemplateFileData != nil {
		args["templateFileData"] = applicationVersion.TemplateFileData
	}

	row, err := db.Query(ctx,
		`INSERT INTO ApplicationVersion AS av (name, link_template, application_id, chart_type, chart_name, chart_url,
				chart_version, compose_file_data, values_file_data, template_file_data)
		VALUES (@name, @linkTemplate, @applicationId, @chartType, @chartName, @chartUrl, @chartVersion,
			@composeFileData::bytea, @valuesFileData::bytea, @templateFileData::bytea)
		RETURNING av.id, av.created_at, av.archived_at, av.name, av.link_template, av.chart_type, av.chart_name,
			av.chart_url, av.chart_version, av.values_file_data, av.template_file_data, av.compose_file_data,
			av.application_id`,
		args)
	if err != nil {
		return fmt.Errorf("can not create ApplicationVersion: %w", err)
	} else if result, err := pgx.CollectExactlyOneRow(row, pgx.RowToStructByName[types.ApplicationVersion]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrNotFound
		} else if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.UniqueViolation {
			err = apierrors.ErrAlreadyExists
		}
		return fmt.Errorf("could not scan ApplicationVersion: %w", err)
	} else {
		*applicationVersion = result
		return nil
	}
}

func UpdateApplicationVersion(ctx context.Context, applicationVersion *types.ApplicationVersion) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`UPDATE ApplicationVersion AS av SET name = @name, archived_at = @archivedAt
		WHERE id = @id
		RETURNING `+applicationVersionOutputExpr,
		pgx.NamedArgs{
			"id":         applicationVersion.ID,
			"name":       applicationVersion.Name,
			"archivedAt": applicationVersion.ArchivedAt,
		})
	if err != nil {
		if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.UniqueViolation {
			err = apierrors.ErrAlreadyExists
		}
		return fmt.Errorf("can not update ApplicationVersion: %w", err)
	} else if updated, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ApplicationVersion]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrNotFound
		}
		return fmt.Errorf("could not scan ApplicationVersion: %w", err)
	} else {
		*applicationVersion = updated
		return nil
	}
}

func GetApplicationVersion(ctx context.Context, applicationVersionID uuid.UUID) (*types.ApplicationVersion, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT `+applicationVersionOutputExpr+`
		FROM ApplicationVersion av
		WHERE id = @id`,
		pgx.NamedArgs{"id": applicationVersionID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not get ApplicationVersion: %w", err)
	} else if data, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToStructByName[types.ApplicationVersion]); err != nil {
		if err == pgx.ErrNoRows {
			return nil, apierrors.ErrNotFound
		}
		return nil, err
	} else {
		return &data, nil
	}
}

func UpdateApplicationImage(ctx context.Context, application *types.Application, imageID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	row := db.QueryRow(ctx,
		`UPDATE Application SET image_id = @imageId WHERE id = @id RETURNING image_id`,
		pgx.NamedArgs{"imageId": imageID, "id": application.ID},
	)
	if err := row.Scan(&application.ImageID); err != nil {
		return fmt.Errorf("could not save image id to application: %w", err)
	}
	return nil
}
