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
	customerOrganizationOutputExpr = `
		co.id,
		co.created_at,
		co.organization_id,
		co.image_id,
		co.name,
		co.features
	`
	customerOrganizationWithUsageOutputExpr = customerOrganizationOutputExpr + `,
		count(distinct(oua.user_account_id)) as user_count,
    	count(distinct(dt.id)) as deployment_target_count
	`
)

func ValidateCustomerOrgBelongsToOrg(ctx context.Context, customerOrgID uuid.UUID, orgID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT EXISTS(SELECT 1 FROM CustomerOrganization WHERE id = @customerOrgID AND organization_id = @orgID)",
		pgx.NamedArgs{
			"customerOrgID": customerOrgID,
			"orgID":         orgID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not validate CustomerOrganization belongs to Organization: %w", err)
	}
	exists, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return fmt.Errorf("could not collect result: %w", err)
	}
	if !exists {
		return fmt.Errorf("customer organization does not belong to organization")
	}
	return nil
}

func CreateCustomerOrganization(ctx context.Context, customerOrg *types.CustomerOrganization) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"INSERT INTO CustomerOrganization AS co (organization_id, image_id, name) "+
			"VALUES (@organizationId, @imageId, @name) RETURNING "+customerOrganizationOutputExpr,
		pgx.NamedArgs{
			"organizationId": customerOrg.OrganizationID,
			"imageId":        customerOrg.ImageID,
			"name":           customerOrg.Name,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert CustomerOrganization: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomerOrganization])
	if err != nil {
		return err
	} else {
		*customerOrg = result
		return nil
	}
}

func GetCustomerOrganizationByID(
	ctx context.Context,
	id uuid.UUID,
) (*types.CustomerOrganizationWithUsage, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		fmt.Sprintf(
			`SELECT %v
			FROM CustomerOrganization co
			LEFT JOIN Organization_UserAccount oua ON co.id = oua.customer_organization_id
			LEFT JOIN DeploymentTarget dt ON co.id = dt.customer_organization_id
			WHERE co.id = @id
			GROUP BY %v
			ORDER BY co.name`,
			customerOrganizationWithUsageOutputExpr, customerOrganizationOutputExpr,
		),
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query CustomerOrganization: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomerOrganizationWithUsage])
	if err != nil {
		return nil, fmt.Errorf("could not collect CustomerOrganization: %w", err)
	}
	return &result, nil
}

func GetCustomerOrganizationsByOrganizationID(
	ctx context.Context,
	orgID uuid.UUID,
) ([]types.CustomerOrganizationWithUsage, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		fmt.Sprintf(
			`SELECT %v
			FROM CustomerOrganization co
			LEFT JOIN Organization_UserAccount oua ON co.id = oua.customer_organization_id
			LEFT JOIN DeploymentTarget dt ON co.id = dt.customer_organization_id
			WHERE co.organization_id = @orgId
			GROUP BY %v
			ORDER BY co.name`,
			customerOrganizationWithUsageOutputExpr, customerOrganizationOutputExpr,
		),
		pgx.NamedArgs{"orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query CustomerOrganization: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.CustomerOrganizationWithUsage])
	if err != nil {
		return nil, fmt.Errorf("could not collect CustomerOrganization: %w", err)
	}
	return result, nil
}

func CountCustomerOrganizationsByOrganizationID(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT count(*) FROM CustomerOrganization WHERE organization_id = @organizationId",
		pgx.NamedArgs{"organizationId": organizationID},
	)
	if err != nil {
		return 0, fmt.Errorf("could not query CustomerOrganization: %w", err)
	}
	if count, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int64]); err != nil {
		return 0, fmt.Errorf("could not count CustomerOrganizations: %w", err)
	} else {
		return count, nil
	}
}

func UpdateCustomerOrganization(ctx context.Context, customerOrg *types.CustomerOrganization) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"UPDATE CustomerOrganization AS co SET name = @name, image_id = @imageId, features = @features "+
			"WHERE co.id = @id AND co.organization_id = @organizationId RETURNING "+customerOrganizationOutputExpr,
		pgx.NamedArgs{
			"id":             customerOrg.ID,
			"organizationId": customerOrg.OrganizationID,
			"name":           customerOrg.Name,
			"imageId":        customerOrg.ImageID,
			"features":       customerOrg.Features,
		},
	)
	if err != nil {
		return fmt.Errorf("could not update CustomerOrganization: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomerOrganization])
	if errors.Is(err, pgx.ErrNoRows) {
		return apierrors.ErrNotFound
	} else if err != nil {
		return err
	} else {
		*customerOrg = result
		return nil
	}
}

func DeleteCustomerOrganizationWithID(ctx context.Context, id uuid.UUID, organizationID uuid.UUID) error {
	err := RunTx(ctx, func(ctx context.Context) error {
		db := internalctx.GetDb(ctx)
		if _, err := db.Exec(ctx, "SET CONSTRAINTS deployment_application_entitlement_id_fkey DEFERRED"); err != nil {
			return err
		}

		cmd, err := db.Exec(
			ctx,
			`DELETE FROM CustomerOrganization WHERE id = @id AND organization_id = @organizationId`,
			pgx.NamedArgs{"id": id, "organizationId": organizationID},
		)
		if err != nil {
			return fmt.Errorf("could not delete CustomerOrganization: %w", err)
		}
		if cmd.RowsAffected() == 0 {
			return apierrors.ErrNotFound
		}
		return nil
	})
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.ForeignKeyViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
	}
	return err
}
