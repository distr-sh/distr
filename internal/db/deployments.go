package db

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"text/template"

	"github.com/compose-spec/compose-go/v2/dotenv"
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"gopkg.in/yaml.v3"
)

const (
	deploymentOutputExpr = `
		d.id, d.created_at, d.deployment_target_id, d.release_name, d.application_entitlement_id, d.docker_type,
		d.automatic_application_updates_enabled
	`
	deploymentWithLatestRevisionFromExpr = `
		Deployment d
			JOIN DeploymentRevision dr ON dr.id = d.latest_deployment_revision_id
			JOIN ApplicationVersion av ON dr.application_version_id = av.id
			JOIN Application a ON av.application_id = a.id
			LEFT JOIN DeploymentRevision dr_current ON dr_current.id = d.current_deployment_revision_id
			LEFT JOIN ApplicationVersion av_current ON av_current.id = dr_current.application_version_id
	`
)

var deploymentWithLatestRevisionOutputExpr = deploymentOutputExpr + `,
	dr.application_version_id AS application_version_id,
	` + deploymentValuesYaml.Output("dr") + `,
	` + deploymentEnvFileData.Output("dr") + `,
	dr.values_hash AS values_hash,
	dr.id AS deployment_revision_id,
	dr.created_at AS deployment_revision_created_at,
	dr.force_restart AS force_restart,
	dr.ignore_revision_skew AS ignore_revision_skew,
	CASE WHEN dr.helm_options_timeout IS NOT NULL THEN (
		dr.helm_options_timeout,
		dr.helm_options_wait_strategy,
		dr.helm_options_rollback_on_failure,
		dr.helm_options_cleanup_on_failure,
		dr.helm_options_force_conflicts
	) END AS helm_options,
	a.id AS application_id,
	a.name AS application_name,
	(` + applicationOutputExpr + `) AS application,
	av.name AS application_version_name,
	av.link_template AS application_link_template,
	CASE WHEN dr.status_type IS NOT NULL THEN (
		dr.status_created_at,
		dr.id,
		dr.status_type,
		dr.status_message
	) END AS latest_status,
	d.current_deployment_revision_id AS current_deployment_revision_id,
	CASE WHEN dr_current.status_type IS NOT NULL THEN (
		dr_current.status_created_at,
		dr_current.id,
		dr_current.status_type,
		dr_current.status_message
	) END AS current_status,
	dr_current.application_version_id AS current_application_version_id,
	av_current.name AS current_application_version_name
`

func GetDeployment(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
	orgID uuid.UUID,
	customerOrganizationID *uuid.UUID,
	partnerOrganizationID *uuid.UUID,
) (*types.Deployment, error) {
	db := internalctx.GetDb(ctx)
	isVendor := customerOrganizationID == nil && partnerOrganizationID == nil
	rows, err := db.Query(ctx,
		"SELECT"+deploymentOutputExpr+
			"FROM Deployment d "+
			"INNER JOIN DeploymentTarget dt ON d.deployment_target_id = dt.id "+
			"LEFT JOIN CustomerOrganization co ON dt.customer_organization_id = co.id "+
			"WHERE d.id = @id AND dt.organization_id = @orgId "+
			"AND (@isVendor "+
			"OR dt.customer_organization_id = @customerOrganizationId "+
			"OR co.partner_organization_id = @partnerOrganizationId)",
		pgx.NamedArgs{
			"id":                     id,
			"userId":                 userID,
			"orgId":                  orgID,
			"isVendor":               isVendor,
			"customerOrganizationId": customerOrganizationID,
			"partnerOrganizationId":  partnerOrganizationID,
		})
	if err != nil {
		return nil, fmt.Errorf("failed to query Deployments: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.Deployment])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to get Deployment: %w", err)
	} else {
		return &result, nil
	}
}

func GetDeploymentsForDeploymentTarget(
	ctx context.Context,
	deploymentTargetID uuid.UUID,
) ([]types.DeploymentWithLatestRevision, error) {
	// TODO all these methods also need the orgId criteria
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+deploymentWithLatestRevisionOutputExpr+`
			FROM `+deploymentWithLatestRevisionFromExpr+`
			WHERE d.deployment_target_id = @deploymentTargetId
			ORDER BY d.created_at`,
		pgx.NamedArgs{"deploymentTargetId": deploymentTargetID})
	if err != nil {
		return nil, fmt.Errorf("failed to query Deployments: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.DeploymentWithLatestRevision])
	if err != nil {
		return nil, fmt.Errorf("failed to scan Deployments: %w", err)
	}

	if err := TemplateDeploymentLinks(result); err != nil {
		return nil, fmt.Errorf("failed to template deployment links: %w", err)
	}

	return result, nil
}

// GetDeploymentsWithAutomaticApplicationUpdates returns every deployment of the application that
// has automatic updates enabled, regardless of the version it is on.
func GetDeploymentsWithAutomaticApplicationUpdates(
	ctx context.Context,
	applicationID uuid.UUID,
) ([]types.DeploymentWithLatestRevision, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+deploymentWithLatestRevisionOutputExpr+`
			FROM `+deploymentWithLatestRevisionFromExpr+`
			WHERE a.id = @applicationId AND d.automatic_application_updates_enabled
			ORDER BY d.created_at`,
		pgx.NamedArgs{"applicationId": applicationID})
	if err != nil {
		return nil, fmt.Errorf("failed to query Deployments: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.DeploymentWithLatestRevision])
	if err != nil {
		return nil, fmt.Errorf("failed to scan Deployments: %w", err)
	}
	return result, nil
}

func TemplateApplicationLink(link string, envFileData []byte, valuesYaml []byte) (string, error) {
	if link == "" {
		return "", nil
	}

	parsedEnv, err := dotenv.UnmarshalBytesWithLookup(envFileData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to parse env file: %w", err)
	}

	valuesMap := make(map[string]any)
	if len(valuesYaml) > 0 {
		if err := yaml.Unmarshal(valuesYaml, &valuesMap); err != nil {
			return "", fmt.Errorf("failed to parse values YAML: %w", err)
		}
	}

	data := map[string]any{
		"Env":    parsedEnv,
		"Values": valuesMap,
	}

	tmpl, err := template.New("link").Parse(link)
	if err != nil {
		return "", fmt.Errorf("failed to parse link template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute link template: %w", err)
	}

	return buf.String(), nil
}

func TemplateDeploymentLinks(deployments []types.DeploymentWithLatestRevision) error {
	for i := range deployments {
		templatedLink, err := TemplateApplicationLink(
			deployments[i].ApplicationLinkTemplate,
			deployments[i].EnvFileData,
			deployments[i].ValuesYaml,
		)
		if err != nil {
			continue
		}
		deployments[i].ApplicationLink = templatedLink
	}
	return nil
}

func CreateDeployment(ctx context.Context, request *api.DeploymentRequest) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`INSERT INTO Deployment AS d
			(deployment_target_id, release_name, application_entitlement_id, docker_type,
				automatic_application_updates_enabled)
			VALUES (@deploymentTargetId, @releaseName, @applicationEntitlementId, @dockerType,
				@automaticApplicationUpdatesEnabled)
			RETURNING`+deploymentOutputExpr,
		pgx.NamedArgs{
			"deploymentTargetId":                 request.DeploymentTargetID,
			"releaseName":                        request.ReleaseName,
			"applicationEntitlementId":           request.ApplicationEntitlementID,
			"dockerType":                         request.DockerType,
			"automaticApplicationUpdatesEnabled": util.PtrDerefOrDefault(request.AutomaticApplicationUpdatesEnabled),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to query Deployments: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.Deployment])
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("%w: release name must be unique per deployment target", apierrors.ErrConflict)
		}
		return fmt.Errorf("could not save Deployment: %w", err)
	} else {
		request.DeploymentID = &result.ID
		return nil
	}
}

func UpdateDeploymentEntitlement(ctx context.Context, deployment *types.Deployment) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`UPDATE Deployment AS d
		SET application_entitlement_id = @applicationEntitlementID
		WHERE id = @id
		RETURNING`+deploymentOutputExpr,
		pgx.NamedArgs{
			"id":                       deployment.ID,
			"applicationEntitlementID": deployment.ApplicationEntitlementID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not update Deployment: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.Deployment]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrNotFound
		}
		return fmt.Errorf("could not update Deployment: %w", err)
	} else {
		*deployment = result
		return nil
	}
}

func UpdateDeploymentAutomaticApplicationUpdates(
	ctx context.Context,
	deployment *types.Deployment,
) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`UPDATE Deployment AS d
		SET automatic_application_updates_enabled = @automaticApplicationUpdatesEnabled
		WHERE id = @id
		RETURNING`+deploymentOutputExpr,
		pgx.NamedArgs{
			"id":                                 deployment.ID,
			"automaticApplicationUpdatesEnabled": deployment.AutomaticApplicationUpdatesEnabled,
		},
	)
	if err != nil {
		return fmt.Errorf("could not update Deployment: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.Deployment]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrNotFound
		}
		return fmt.Errorf("could not update Deployment: %w", err)
	} else {
		*deployment = result
		return nil
	}
}

func UpdateDeploymentUnsetEntitlementIDWithOrganizationID(ctx context.Context, organizationID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(
		ctx,
		`UPDATE Deployment
		SET application_entitlement_id = NULL
		WHERE deployment_target_id IN (
			SELECT id FROM DeploymentTarget WHERE organization_id = @organizationID
		)`,
		pgx.NamedArgs{"organizationID": organizationID},
	)
	if err != nil {
		return fmt.Errorf("could not update Deployment: %w", err)
	}
	return nil
}

func UpdateDeploymentUnsetEntitlementIDWithOrganizationSubscriptionType(
	ctx context.Context,
	subscriptionType []types.SubscriptionType,
) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(
		ctx,
		`UPDATE Deployment
		SET application_entitlement_id = NULL
		WHERE deployment_target_id IN (
			SELECT dt.id FROM DeploymentTarget dt
				JOIN Organization o ON dt.organization_id = o.id
			WHERE o.subscription_type = ANY(@subscriptionType)
		)`,
		pgx.NamedArgs{"subscriptionType": subscriptionType},
	)
	if err != nil {
		return fmt.Errorf("could not update Deployment: %w", err)
	}
	return nil
}

func DeleteDeploymentWithID(ctx context.Context, id uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	res, err := db.Exec(ctx, "DELETE FROM Deployment WHERE id = @id", pgx.NamedArgs{"id": id})
	if err == nil && res.RowsAffected() == 0 {
		err = apierrors.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("could not delete Deployment: %w", err)
	}
	return nil
}

func CreateDeploymentRevision(ctx context.Context, request *api.DeploymentRequest) (*types.DeploymentRevision, error) {
	deploymentID := dbcrypto.ScopeOf(request.DeploymentID)
	valuesYamlEnc, err := deploymentValuesYaml.EncryptBytes(request.ValuesYaml, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("could not encrypt deployment values: %w", err)
	}
	envFileDataEnc, err := deploymentEnvFileData.EncryptBytes(request.EnvFileData, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("could not encrypt deployment env file: %w", err)
	}
	db := internalctx.GetDb(ctx)
	args := pgx.NamedArgs{
		"deploymentId":           request.DeploymentID,
		"applicationVersionId":   request.ApplicationVersionID,
		"valuesYamlEnc":          valuesYamlEnc,
		"envFileDataEnc":         envFileDataEnc,
		"valuesHash":             request.ValuesHash,
		"forceRestart":           request.ForceRestart,
		"ignoreRevisionSkew":     request.IgnoreRevisionSkew,
		"createdByUserAccountId": request.CreatedByUserAccountID,
		"trigger":                request.Trigger,
	}

	if request.HelmOptions != nil {
		maps.Copy(args, pgx.NamedArgs{
			"helmOptionsTimeout":           request.HelmOptions.Timeout,
			"helmOptionsWaitStrategy":      request.HelmOptions.WaitStrategy,
			"helmOptionsRollbackOnFailure": request.HelmOptions.RollbackOnFailure,
			"helmOptionsCleanupOnFailure":  request.HelmOptions.CleanupOnFailure,
			"helmOptionsForceConflicts":    request.HelmOptions.ForceConflicts,
		})
	}

	rows, err := db.Query(
		ctx,
		`WITH inserted AS (
			INSERT INTO DeploymentRevision AS dr (
				deployment_id,
				application_version_id,
				values_yaml_enc,
				env_file_data_enc,
				values_hash,
				force_restart,
				ignore_revision_skew,
				helm_options_timeout,
				helm_options_wait_strategy,
				helm_options_rollback_on_failure,
				helm_options_cleanup_on_failure,
				helm_options_force_conflicts,
				created_by_user_account_id,
				trigger
			) VALUES (
				@deploymentId,
				@applicationVersionId,
				@valuesYamlEnc,
				@envFileDataEnc,
				@valuesHash,
				@forceRestart,
				@ignoreRevisionSkew,
				@helmOptionsTimeout,
				@helmOptionsWaitStrategy,
				@helmOptionsRollbackOnFailure,
				@helmOptionsCleanupOnFailure,
				@helmOptionsForceConflicts,
				@createdByUserAccountId,
				@trigger
			) RETURNING
				dr.id,
				dr.created_at,
				dr.deployment_id,
				dr.application_version_id,
				dr.values_hash,
				dr.force_restart,
				dr.ignore_revision_skew,
				dr.created_by_user_account_id,
				dr.trigger,
				CASE WHEN dr.helm_options_timeout IS NOT NULL THEN (
					dr.helm_options_timeout,
					dr.helm_options_wait_strategy,
					dr.helm_options_rollback_on_failure,
					dr.helm_options_cleanup_on_failure,
					dr.helm_options_force_conflicts
				) END as helm_options
		), updated AS (
			UPDATE Deployment
			SET latest_deployment_revision_id = (SELECT id FROM inserted)
			WHERE id = @deploymentId
		)
		SELECT * FROM inserted`,
		args,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query DeploymentRevision: %w", err)
	}

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.DeploymentRevision])
	if err != nil {
		return nil, fmt.Errorf("could not save DeploymentRevision: %w", err)
	} else {
		return &result, nil
	}
}

// UpdateDeploymentCurrentRevision marks the given revision as the one currently applied on the
// deployment target. It never moves the current revision back to an older one, because an agent may
// report a status of the previous revision after the next one has already been applied. Reporting
// the revision that is already current writes nothing, which is what every agent does once per
// AGENT_INTERVAL for as long as nothing changes.
func UpdateDeploymentCurrentRevision(ctx context.Context, revisionID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	if _, err := db.Exec(
		ctx,
		`UPDATE Deployment d
		SET current_deployment_revision_id = dr.id
		FROM DeploymentRevision dr
		WHERE dr.id = @revisionId
			AND d.id = dr.deployment_id
			AND (
				d.current_deployment_revision_id IS NULL
				OR dr.created_at > (
					SELECT created_at FROM DeploymentRevision WHERE id = d.current_deployment_revision_id
				)
			)`,
		pgx.NamedArgs{"revisionId": revisionID},
	); err != nil {
		return fmt.Errorf("could not update current Deployment revision: %w", err)
	}
	return nil
}

// GetLatestDeploymentRevisionIDs returns the IDs of the most recently created revisions
// of the given deployment, newest first.
func GetLatestDeploymentRevisionIDs(ctx context.Context, deploymentID uuid.UUID, limit int) ([]uuid.UUID, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT id FROM DeploymentRevision
		WHERE deployment_id = @deploymentId
		ORDER BY created_at DESC
		LIMIT @limit`,
		pgx.NamedArgs{"deploymentId": deploymentID, "limit": limit},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query DeploymentRevision: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("could not collect DeploymentRevision: %w", err)
	}
	return result, nil
}

func GetDeploymentRevisions(
	ctx context.Context,
	deploymentID uuid.UUID,
) ([]types.DeploymentRevisionWithCreator, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT
				dr.id,
				dr.created_at,
				dr.application_version_id,
				av.name AS application_version_name,
				d.release_name AS release_name,
				d.docker_type AS docker_type,
				`+deploymentValuesYaml.Output("dr")+`,
				`+deploymentEnvFileData.Output("dr")+`,
				dr.force_restart AS force_restart,
				dr.ignore_revision_skew AS ignore_revision_skew,
				CASE WHEN dr.helm_options_timeout IS NOT NULL THEN (
					dr.helm_options_timeout,
					dr.helm_options_wait_strategy,
					dr.helm_options_rollback_on_failure,
					dr.helm_options_cleanup_on_failure,
					dr.helm_options_force_conflicts
				) END AS helm_options,
				dr.trigger AS trigger,
				u.id AS created_by_id,
				u.name AS created_by_name,
				u.email AS created_by_email,
				u.image_id AS created_by_image_id,
				j.customer_organization_id AS created_by_customer_organization_id,
				j.partner_organization_id AS created_by_partner_organization_id,
				(dr.created_by_user_account_id IS NOT NULL AND j.user_account_id IS NULL)
					AS created_by_deleted,
				CASE WHEN dr.status_type IS NOT NULL THEN (
					dr.status_created_at,
					dr.id,
					dr.status_type,
					dr.status_message
				) END AS latest_status
			FROM DeploymentRevision dr
				JOIN Deployment d ON dr.deployment_id = d.id
				JOIN DeploymentTarget dt ON d.deployment_target_id = dt.id
				JOIN ApplicationVersion av ON dr.application_version_id = av.id
				LEFT JOIN UserAccount u ON dr.created_by_user_account_id = u.id
				LEFT JOIN Organization_UserAccount j
					ON j.user_account_id = u.id AND j.organization_id = dt.organization_id
			WHERE dr.deployment_id = @deploymentId
			ORDER BY dr.created_at DESC`,
		pgx.NamedArgs{"deploymentId": deploymentID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query DeploymentRevisions: %w", err)
	}

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.DeploymentRevisionWithCreator])
	if err != nil {
		return nil, fmt.Errorf("failed to scan DeploymentRevisions: %w", err)
	}

	return result, nil
}

func GetDeploymentIDForRevisionID(ctx context.Context, revisionID uuid.UUID) (uuid.UUID, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		"SELECT deployment_id from DeploymentRevision WHERE id = @revisionId",
		pgx.NamedArgs{"revisionId": revisionID},
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to query Deployment ID: %w", err)
	}

	deploymentID, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to scan Deployment ID: %w", err)
	}

	return deploymentID, nil
}
