package db

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const customerOrganizationLinkOutputExpr = `
	col.id,
	col.created_at,
	col.customer_organization_id,
	col.name,
	col.link
`

func GetCustomerOrganizationLinks(
	ctx context.Context,
	customerOrganizationID uuid.UUID,
) ([]types.CustomerOrganizationLink, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		fmt.Sprintf(
			`SELECT %v
			FROM CustomerOrganizationLink col
			WHERE col.customer_organization_id = @customerOrganizationId
			ORDER BY col.name ASC`,
			customerOrganizationLinkOutputExpr,
		),
		pgx.NamedArgs{"customerOrganizationId": customerOrganizationID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query CustomerOrganizationLinks: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.CustomerOrganizationLink])
	if err != nil {
		return nil, fmt.Errorf("could not collect CustomerOrganizationLinks: %w", err)
	}
	return result, nil
}

func CreateCustomerOrganizationLink(
	ctx context.Context,
	customerOrganizationID uuid.UUID,
	name string,
	link string,
) (*types.CustomerOrganizationLink, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`INSERT INTO CustomerOrganizationLink AS col (customer_organization_id, name, link)
		VALUES (@customerOrganizationId, @name, @link)
		RETURNING `+customerOrganizationLinkOutputExpr,
		pgx.NamedArgs{
			"customerOrganizationId": customerOrganizationID,
			"name":                   name,
			"link":                   link,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not insert CustomerOrganizationLink: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomerOrganizationLink])
	if err != nil {
		return nil, fmt.Errorf("could not collect CustomerOrganizationLink: %w", err)
	}
	return &result, nil
}

func UpdateCustomerOrganizationLink(
	ctx context.Context,
	id uuid.UUID,
	customerOrganizationID uuid.UUID,
	name string,
	link string,
) (*types.CustomerOrganizationLink, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`UPDATE CustomerOrganizationLink AS col
		SET name = @name, link = @link
		WHERE col.id = @id AND col.customer_organization_id = @customerOrganizationId
		RETURNING `+customerOrganizationLinkOutputExpr,
		pgx.NamedArgs{
			"id":                     id,
			"customerOrganizationId": customerOrganizationID,
			"name":                   name,
			"link":                   link,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not update CustomerOrganizationLink: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomerOrganizationLink])
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not collect CustomerOrganizationLink: %w", err)
	}
	return &result, nil
}

func DeleteCustomerOrganizationLink(
	ctx context.Context,
	id uuid.UUID,
	customerOrganizationID uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		`DELETE FROM CustomerOrganizationLink
		WHERE id = @id AND customer_organization_id = @customerOrganizationId`,
		pgx.NamedArgs{
			"id":                     id,
			"customerOrganizationId": customerOrganizationID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not delete CustomerOrganizationLink: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}
