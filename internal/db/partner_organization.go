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

const (
	partnerOrganizationOutputExpr = `
		po.id,
		po.created_at,
		po.organization_id,
		po.name
	`
	partnerOrganizationWithUsageOutputExpr = partnerOrganizationOutputExpr + `,
		count(distinct(oua.user_account_id)) as user_count,
		count(distinct(co.id)) as customer_organization_count
	`
)

func ValidatePartnerOrgBelongsToOrg(ctx context.Context, partnerOrgID uuid.UUID, orgID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT EXISTS(SELECT 1 FROM PartnerOrganization WHERE id = @partnerOrgID AND organization_id = @orgID)",
		pgx.NamedArgs{
			"partnerOrgID": partnerOrgID,
			"orgID":        orgID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not validate PartnerOrganization belongs to Organization: %w", err)
	}
	exists, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return fmt.Errorf("could not collect result: %w", err)
	}
	if !exists {
		return fmt.Errorf("partner organization does not belong to organization")
	}
	return nil
}

func CreatePartnerOrganization(ctx context.Context, partnerOrg *types.PartnerOrganization) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"INSERT INTO PartnerOrganization AS po (organization_id, name) "+
			"VALUES (@organizationId, @name) RETURNING "+partnerOrganizationOutputExpr,
		pgx.NamedArgs{
			"organizationId": partnerOrg.OrganizationID,
			"name":           partnerOrg.Name,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert PartnerOrganization: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.PartnerOrganization])
	if err != nil {
		return err
	}
	*partnerOrg = result
	return nil
}

func GetPartnerOrganizationByID(
	ctx context.Context,
	id uuid.UUID,
) (*types.PartnerOrganizationWithUsage, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		fmt.Sprintf(
			`SELECT %v
			FROM PartnerOrganization po
			LEFT JOIN Organization_UserAccount oua ON po.id = oua.partner_organization_id
			LEFT JOIN CustomerOrganization co ON po.id = co.partner_organization_id
			WHERE po.id = @id
			GROUP BY %v`,
			partnerOrganizationWithUsageOutputExpr, partnerOrganizationOutputExpr,
		),
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query PartnerOrganization: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.PartnerOrganizationWithUsage])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not collect PartnerOrganization: %w", err)
	}
	return &result, nil
}

func GetPartnerOrganizationsByOrganizationID(
	ctx context.Context,
	orgID uuid.UUID,
) ([]types.PartnerOrganizationWithUsage, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		fmt.Sprintf(
			`SELECT %v
			FROM PartnerOrganization po
			LEFT JOIN Organization_UserAccount oua ON po.id = oua.partner_organization_id
			LEFT JOIN CustomerOrganization co ON po.id = co.partner_organization_id
			WHERE po.organization_id = @orgId
			GROUP BY %v
			ORDER BY po.name`,
			partnerOrganizationWithUsageOutputExpr, partnerOrganizationOutputExpr,
		),
		pgx.NamedArgs{"orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query PartnerOrganization: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.PartnerOrganizationWithUsage])
	if err != nil {
		return nil, fmt.Errorf("could not collect PartnerOrganization: %w", err)
	}
	return result, nil
}

func UpdatePartnerOrganization(ctx context.Context, partnerOrg *types.PartnerOrganization) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"UPDATE PartnerOrganization AS po SET name = @name "+
			"WHERE po.id = @id AND po.organization_id = @organizationId RETURNING "+partnerOrganizationOutputExpr,
		pgx.NamedArgs{
			"id":             partnerOrg.ID,
			"organizationId": partnerOrg.OrganizationID,
			"name":           partnerOrg.Name,
		},
	)
	if err != nil {
		return fmt.Errorf("could not update PartnerOrganization: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.PartnerOrganization])
	if errors.Is(err, pgx.ErrNoRows) {
		return apierrors.ErrNotFound
	} else if err != nil {
		return err
	}
	*partnerOrg = result
	return nil
}

func DeletePartnerOrganizationWithID(ctx context.Context, id uuid.UUID, organizationID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		`DELETE FROM PartnerOrganization WHERE id = @id AND organization_id = @organizationId`,
		pgx.NamedArgs{"id": id, "organizationId": organizationID},
	)
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.ForeignKeyViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return err
	}
	if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}
