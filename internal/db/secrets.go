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

const secretOutputExpr = `
	s.id,
	s.created_at,
	s.updated_at,
	s.updated_by_useraccount_id,
	s.organization_id,
	s.customer_organization_id,
	s.key,
	s.value`

const secretWithUpdatedByOutputExpr = secretOutputExpr + `,
	CASE WHEN u.id IS NULL
		THEN NULL
		ELSE (` + userAccountOutputExpr + `)
	END AS updated_by`

func GetSecrets(
	ctx context.Context,
	organizationID uuid.UUID,
	customerOrganizationID,
	partnerOrganizationID *uuid.UUID,
) ([]types.SecretWithUpdatedBy, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT `+secretWithUpdatedByOutputExpr+` FROM Secret s
		LEFT JOIN UserAccount u ON s.updated_by_useraccount_id = u.id
		LEFT JOIN CustomerOrganization c ON s.customer_organization_id = c.id
		WHERE s.organization_id = @organizationID
			AND (@isVendor
				OR s.customer_organization_id = @customerOrganizationID
				OR c.partner_organization_id = @partnerOrganizationID)
		ORDER BY s.key ASC`,
		pgx.NamedArgs{
			"organizationID":         organizationID,
			"customerOrganizationID": customerOrganizationID,
			"partnerOrganizationID":  partnerOrganizationID,
			"isVendor":               customerOrganizationID == nil && partnerOrganizationID == nil,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query Secret: %w", err)
	}

	if secrets, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.SecretWithUpdatedBy]); err != nil {
		return nil, fmt.Errorf("failed to collect Secret: %w", err)
	} else {
		return secrets, nil
	}
}

func GetSecretsForDeploymentTarget(
	ctx context.Context,
	dt types.DeploymentTarget,
) ([]types.SecretWithUpdatedBy, error) {
	return GetSecretsByScope(ctx, dt.OrganizationID, dt.CustomerOrganizationID)
}

func GetSecretsByScope(
	ctx context.Context,
	organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) ([]types.SecretWithUpdatedBy, error) {
	if customerOrganizationID != nil {
		return GetSecretsForCustomer(ctx, *customerOrganizationID)
	} else {
		return GetSecretsForOrganization(ctx, organizationID)
	}
}

func GetSecretsForOrganization(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]types.SecretWithUpdatedBy, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT `+secretWithUpdatedByOutputExpr+` FROM Secret s
		LEFT JOIN UserAccount u
			ON s.updated_by_useraccount_id = u.id
		WHERE s.organization_id = @organization_id
			AND s.customer_organization_id IS NULL
		ORDER BY s.key ASC`,
		pgx.NamedArgs{
			"organization_id": organizationID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query Secret: %w", err)
	}

	if secrets, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.SecretWithUpdatedBy]); err != nil {
		return nil, fmt.Errorf("failed to collect Secret: %w", err)
	} else {
		return secrets, nil
	}
}

func GetSecretsForCustomer(
	ctx context.Context,
	customerOrganizationID uuid.UUID,
) ([]types.SecretWithUpdatedBy, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT `+secretWithUpdatedByOutputExpr+` FROM Secret s
		LEFT JOIN UserAccount u
			ON s.updated_by_useraccount_id = u.id
		WHERE s.customer_organization_id = @customer_organization_id
		ORDER BY s.key ASC`,
		pgx.NamedArgs{
			"customer_organization_id": customerOrganizationID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query Secret: %w", err)
	}

	if secrets, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.SecretWithUpdatedBy]); err != nil {
		return nil, fmt.Errorf("failed to collect Secret: %w", err)
	} else {
		return secrets, nil
	}
}

func GetSecretByID(
	ctx context.Context,
	id uuid.UUID,
	orgID uuid.UUID,
	customerOrganizationID, partnerOrganizationID *uuid.UUID,
) (*types.SecretWithUpdatedBy, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT `+secretWithUpdatedByOutputExpr+` FROM Secret s
		LEFT JOIN UserAccount u ON s.updated_by_useraccount_id = u.id
		LEFT JOIN CustomerOrganization c ON s.customer_organization_id = c.id
		WHERE s.id = @id
			AND s.organization_id = @organizationID
			AND (@isVendor
				OR s.customer_organization_id = @customerOrganizationID
				OR c.partner_organization_id = @partnerOrganizationID)`,
		pgx.NamedArgs{
			"id":                     id,
			"organizationID":         orgID,
			"customerOrganizationID": customerOrganizationID,
			"partnerOrganizationID":  partnerOrganizationID,
			"isVendor":               customerOrganizationID == nil && partnerOrganizationID == nil,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query Secret: %w", err)
	}

	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.SecretWithUpdatedBy]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to collect Secret: %w", err)
	} else {
		return &result, nil
	}
}

func CreateSecret(
	ctx context.Context,
	organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
	updatedByUserAccountID uuid.UUID,
	key, value string,
) (*types.SecretWithUpdatedBy, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`WITH inserted AS (
			INSERT INTO Secret (key, value, organization_id, customer_organization_id, updated_by_useraccount_id)
			VALUES (@key, @value, @organization_id, @customer_organization_id, @updated_by_useraccount_id)
			RETURNING *
		)
		SELECT `+secretWithUpdatedByOutputExpr+` FROM inserted s
		LEFT JOIN UserAccount u
			ON s.updated_by_useraccount_id = u.id
		`,
		pgx.NamedArgs{
			"key":                       key,
			"organization_id":           organizationID,
			"customer_organization_id":  customerOrganizationID,
			"updated_by_useraccount_id": updatedByUserAccountID,
			"value":                     value,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query Secret: %w", err)
	}

	if secret, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.SecretWithUpdatedBy]); err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.Code == pgerrcode.UniqueViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return nil, fmt.Errorf("failed to collect Secret: %w", err)
	} else {
		return &secret, nil
	}
}

func UpdateSecret(ctx context.Context,
	id uuid.UUID,
	customerOrganizationID *uuid.UUID,
	updatedByUserAccountID uuid.UUID,
	value string,
) (*types.SecretWithUpdatedBy, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`WITH updated AS (
			UPDATE Secret SET
				updated_by_useraccount_id = @updated_by_useraccount_id,
				updated_at = NOW(),
				value = @value
			WHERE id = @id
				AND (@is_vendor OR customer_organization_id = @customer_organization_id)
			RETURNING *
		)
		SELECT `+secretWithUpdatedByOutputExpr+` FROM updated s
		LEFT JOIN UserAccount u
			ON s.updated_by_useraccount_id = u.id`,
		pgx.NamedArgs{
			"id":                        id,
			"customer_organization_id":  customerOrganizationID,
			"is_vendor":                 customerOrganizationID == nil,
			"updated_by_useraccount_id": updatedByUserAccountID,
			"value":                     value,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query Secret: %w", err)
	}

	if secret, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.SecretWithUpdatedBy]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to collect Secret: %w", err)
	} else {
		return &secret, nil
	}
}

func DeleteSecret(
	ctx context.Context,
	id uuid.UUID,
	organizationID uuid.UUID,
	customerOrganizationID, partnerOrganizationID *uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(
		ctx,
		`DELETE FROM Secret
		WHERE id = @id
			AND organization_id = @organizationID
			AND (@isVendor
				OR customer_organization_id = @customerOrganizationID
				OR EXISTS (
					SELECT 1 FROM CustomerOrganization c
					WHERE c.id = customer_organization_id
					AND c.partner_organization_id = @partnerOrganizationID
				))`,
		pgx.NamedArgs{
			"id":                     id,
			"organizationID":         organizationID,
			"customerOrganizationID": customerOrganizationID,
			"partnerOrganizationID":  partnerOrganizationID,
			"isVendor":               customerOrganizationID == nil && partnerOrganizationID == nil,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to delete Secret: %w", err)
	} else if cmd.RowsAffected() == 0 {
		return fmt.Errorf("failed to delete Secret: %w", apierrors.ErrNotFound)
	} else {
		return nil
	}
}
