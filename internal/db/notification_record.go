package db

import (
	"context"
	"errors"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const notificationRecordOutputExpr = `
	r.id,
	r.created_at,
	r.organization_id,
	r.customer_organization_id,
	r.user_account_id,
	r.source_type,
	r.source_configuration_id,
	r.subject_id,
	r.type,
	r.deployment_revision_id,
	r.resolved_at,
	r.details,
	r.delivery_error `

// ErrNotificationRecordExists is returned when a record for the same recipient and subject has
// been written already, which is what keeps a recipient from hearing about the same version twice,
// including when a vendor and a customer configuration both cover them.
var ErrNotificationRecordExists = errors.New("notification record already exists")

func SaveNotificationRecord(ctx context.Context, record *types.NotificationRecord) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`WITH inserted AS (
			INSERT INTO NotificationRecord (
				organization_id,
				customer_organization_id,
				user_account_id,
				source_type,
				source_configuration_id,
				subject_id,
				type,
				deployment_revision_id,
				details,
				delivery_error
			)
			VALUES (
				@organizationID,
				@customerOrganizationID,
				@userAccountID,
				@sourceType,
				@sourceConfigurationID,
				@subjectID,
				@type,
				@deploymentRevisionID,
				@details,
				@deliveryError
			)
			RETURNING *
		)
		SELECT`+notificationRecordOutputExpr+`FROM inserted r`,
		pgx.NamedArgs{
			"organizationID":         record.OrganizationID,
			"customerOrganizationID": record.CustomerOrganizationID,
			"userAccountID":          record.UserAccountID,
			"sourceType":             record.SourceType,
			"sourceConfigurationID":  record.SourceConfigurationID,
			"subjectID":              record.SubjectID,
			"type":                   record.Type,
			"deploymentRevisionID":   record.DeploymentRevisionID,
			"details":                record.Details,
			"deliveryError":          record.DeliveryError,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to save NotificationRecord: %w", err)
	}

	// The unique violation surfaces when the row is read, not when the query is sent.
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.NotificationRecord])
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == pgerrcode.UniqueViolation {
		return ErrNotificationRecordExists
	} else if err != nil {
		return fmt.Errorf("failed to collect NotificationRecord: %w", err)
	}
	*record = result

	return nil
}

func HasOpenStaleWarning(ctx context.Context, configID, deploymentID uuid.UUID) (bool, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT EXISTS (
			SELECT 1 FROM NotificationRecord r
			JOIN DeploymentRevision dr ON dr.id = r.deployment_revision_id
			WHERE r.source_type = 'alert'
				AND r.source_configuration_id = @alertConfigurationID
				AND dr.deployment_id = @deploymentID
				AND r.type = 'warning'
				AND r.resolved_at IS NULL
		)`,
		pgx.NamedArgs{"alertConfigurationID": configID, "deploymentID": deploymentID},
	)
	if err != nil {
		return false, fmt.Errorf("failed to query open stale warning: %w", err)
	}

	exists, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return false, fmt.Errorf("failed to collect open stale warning: %w", err)
	}
	return exists, nil
}

// ResolveStaleWarnings resolves the open stale warnings of every alert configuration for the deployment and returns
// the configurations they belonged to. Only one caller can resolve a warning, so it is the one that has to send the
// recovery notification.
func ResolveStaleWarnings(ctx context.Context, deploymentID uuid.UUID) ([]uuid.UUID, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`WITH resolved AS (
			UPDATE NotificationRecord r SET resolved_at = now()
			FROM DeploymentRevision dr
			WHERE dr.id = r.deployment_revision_id
				AND dr.deployment_id = @deploymentID
				AND r.source_type = 'alert'
				AND r.type = 'warning'
				AND r.resolved_at IS NULL
			RETURNING r.source_configuration_id
		)
		SELECT DISTINCT source_configuration_id FROM resolved WHERE source_configuration_id IS NOT NULL`,
		pgx.NamedArgs{"deploymentID": deploymentID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve stale warnings: %w", err)
	}

	configIDs, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("failed to collect resolved stale warnings: %w", err)
	}
	return configIDs, nil
}

func NotificationRecordExists(ctx context.Context, userAccountID, subjectID uuid.UUID) (bool, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT EXISTS (
			SELECT 1 FROM NotificationRecord r
			WHERE r.user_account_id = @userAccountID AND r.subject_id = @subjectID
		)`,
		pgx.NamedArgs{
			"userAccountID": userAccountID,
			"subjectID":     subjectID,
		},
	)
	if err != nil {
		return false, fmt.Errorf("failed to query NotificationRecord: %w", err)
	}

	return pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
}

func GetNotificationRecords(
	ctx context.Context,
	organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) ([]types.NotificationRecord, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(
		ctx,
		// A record of an update notification carries the recipient's customer organization, so a
		// vendor has to see the rows of its customers too in order to see what was sent at all.
		`SELECT`+notificationRecordOutputExpr+`FROM NotificationRecord r
		WHERE r.organization_id = @organizationID
			AND (@isVendor OR r.customer_organization_id = @customerOrganizationID)
		ORDER BY r.created_at DESC`,
		pgx.NamedArgs{
			"organizationID":         organizationID,
			"customerOrganizationID": customerOrganizationID,
			"isVendor":               customerOrganizationID == nil,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query NotificationRecord: %w", err)
	}

	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.NotificationRecord])
	if err != nil {
		return nil, fmt.Errorf("failed to collect NotificationRecord: %w", err)
	}

	return records, nil
}
