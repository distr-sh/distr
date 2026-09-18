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
	r.details,
	r.message `

// ErrNotificationRecordExists is returned when a record for the same configuration, recipient and
// subject has been written already, which is what keeps a recipient from hearing about the same
// version twice.
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
				details,
				message
			)
			VALUES (
				@organizationID,
				@customerOrganizationID,
				@userAccountID,
				@sourceType,
				@sourceConfigurationID,
				@subjectID,
				@type,
				@details,
				@message
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
			"details":                record.Details,
			"message":                record.Message,
		},
	)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == pgerrcode.UniqueViolation {
			return ErrNotificationRecordExists
		}
		return fmt.Errorf("failed to save NotificationRecord: %w", err)
	}

	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.NotificationRecord]); err != nil {
		return fmt.Errorf("failed to collect NotificationRecord: %w", err)
	} else {
		*record = result
	}

	return nil
}

func GetLatestNotificationRecord(
	ctx context.Context,
	configID, previousID uuid.UUID,
) (*types.NotificationRecord, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+notificationRecordOutputExpr+`FROM NotificationRecord r
		WHERE r.source_configuration_id = @sourceConfigurationID
			AND r.details ->> 'previousDeploymentRevisionStatusId' = @previousDeploymentStatusID
		ORDER BY r.created_at DESC LIMIT 1`,
		pgx.NamedArgs{
			"sourceConfigurationID":      configID,
			"previousDeploymentStatusID": previousID.String(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query NotificationRecord exists: %w", err)
	}

	if record, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.NotificationRecord]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to collect NotificationRecord exists: %w", err)
	} else {
		return &record, nil
	}
}

func NotificationRecordExists(
	ctx context.Context,
	configID, userAccountID, subjectID uuid.UUID,
) (bool, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT count(*) > 0 FROM NotificationRecord r
		WHERE r.source_configuration_id = @sourceConfigurationID
			AND r.user_account_id = @userAccountID
			AND r.subject_id = @subjectID`,
		pgx.NamedArgs{
			"sourceConfigurationID": configID,
			"userAccountID":         userAccountID,
			"subjectID":             subjectID,
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
		`SELECT`+notificationRecordOutputExpr+`FROM NotificationRecord r
		WHERE r.organization_id = @organizationID
			AND ((@isVendor AND r.customer_organization_id IS NULL)
				OR r.customer_organization_id = @customerOrganizationID)
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
