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

const sidebarLinkOutputExpr = `
	sl.id,
	sl.created_at,
	sl.organization_id,
	sl.customer_organization_id,
	sl.name,
	sl.link
`

func GetSidebarLinks(
	ctx context.Context,
	customerOrganizationID uuid.UUID,
) ([]types.SidebarLink, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		fmt.Sprintf(
			`SELECT %v
			FROM SidebarLink sl
			WHERE sl.customer_organization_id = @customerOrganizationId
			ORDER BY sl.name ASC`,
			sidebarLinkOutputExpr,
		),
		pgx.NamedArgs{"customerOrganizationId": customerOrganizationID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query SidebarLinks: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.SidebarLink])
	if err != nil {
		return nil, fmt.Errorf("could not collect SidebarLinks: %w", err)
	}
	return result, nil
}

func CreateSidebarLink(
	ctx context.Context,
	organizationID uuid.UUID,
	customerOrganizationID uuid.UUID,
	name string,
	link string,
) (*types.SidebarLink, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`INSERT INTO SidebarLink AS sl (organization_id, customer_organization_id, name, link)
		VALUES (@organizationId, @customerOrganizationId, @name, @link)
		RETURNING `+sidebarLinkOutputExpr,
		pgx.NamedArgs{
			"organizationId":         organizationID,
			"customerOrganizationId": customerOrganizationID,
			"name":                   name,
			"link":                   link,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not insert SidebarLink: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.SidebarLink])
	if err != nil {
		return nil, fmt.Errorf("could not collect SidebarLink: %w", err)
	}
	return &result, nil
}

func UpdateSidebarLink(
	ctx context.Context,
	id uuid.UUID,
	customerOrganizationID uuid.UUID,
	name string,
	link string,
) (*types.SidebarLink, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`UPDATE SidebarLink AS sl
		SET name = @name, link = @link
		WHERE sl.id = @id AND sl.customer_organization_id = @customerOrganizationId
		RETURNING `+sidebarLinkOutputExpr,
		pgx.NamedArgs{
			"id":                     id,
			"customerOrganizationId": customerOrganizationID,
			"name":                   name,
			"link":                   link,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not update SidebarLink: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.SidebarLink])
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not collect SidebarLink: %w", err)
	}
	return &result, nil
}

func DeleteSidebarLink(
	ctx context.Context,
	id uuid.UUID,
	customerOrganizationID uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		`DELETE FROM SidebarLink
		WHERE id = @id AND customer_organization_id = @customerOrganizationId`,
		pgx.NamedArgs{
			"id":                     id,
			"customerOrganizationId": customerOrganizationID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not delete SidebarLink: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}
