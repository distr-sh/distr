package db

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func CreateApplicationVersionResources(
	ctx context.Context,
	versionID uuid.UUID,
	resources []types.ApplicationVersionResource,
) error {
	if len(resources) == 0 {
		return nil
	}
	db := internalctx.GetDb(ctx)
	_, err := db.CopyFrom(
		ctx,
		pgx.Identifier{"applicationversionresource"},
		[]string{"application_version_id", "name", "content", "visible_to_customers"},
		pgx.CopyFromSlice(len(resources), func(i int) ([]any, error) {
			return []any{versionID, resources[i].Name, resources[i].Content, resources[i].VisibleToCustomers}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("could not create ApplicationVersionResources: %w", err)
	}
	return nil
}

func GetApplicationVersionResources(
	ctx context.Context,
	versionID uuid.UUID,
) ([]types.ApplicationVersionResource, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT id, application_version_id, name, content, visible_to_customers
		FROM ApplicationVersionResource
		WHERE application_version_id = @versionId
		ORDER BY name`,
		pgx.NamedArgs{"versionId": versionID})
	if err != nil {
		return nil, fmt.Errorf("could not query ApplicationVersionResources: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ApplicationVersionResource])
	if err != nil {
		return nil, fmt.Errorf("could not scan ApplicationVersionResources: %w", err)
	}
	return result, nil
}

func GetApplicationVersionResourcesVisibleToCustomers(
	ctx context.Context,
	versionID uuid.UUID,
) ([]types.ApplicationVersionResource, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT id, application_version_id, name, content, visible_to_customers
		FROM ApplicationVersionResource
		WHERE application_version_id = @versionId AND visible_to_customers = true
		ORDER BY name`,
		pgx.NamedArgs{"versionId": versionID})
	if err != nil {
		return nil, fmt.Errorf("could not query ApplicationVersionResources: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ApplicationVersionResource])
	if err != nil {
		return nil, fmt.Errorf("could not scan ApplicationVersionResources: %w", err)
	}
	return result, nil
}
