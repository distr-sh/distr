package db

import (
	"context"
	"errors"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/buildconfig"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func CreateControllerVersion(ctx context.Context) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(ctx,
		`INSERT INTO ControllerVersion (name, manifest_file_revision, compose_file_revision)
			VALUES (@name, @manifestRevision, @composeRevision)
		ON CONFLICT (name) DO UPDATE SET
			manifest_file_revision = @manifestRevision,
			compose_file_revision = @composeRevision`,
		pgx.NamedArgs{
			"name":             buildconfig.Version(),
			"manifestRevision": types.CurrentManifestFileRevision,
			"composeRevision":  types.CurrentComposeFileRevision,
		})
	return err
}

// ApplyAutomaticControllerUpdates points every DeploymentTarget with automatic updates enabled at the controller
// version of the running server. Controllers self-update as soon as the version they receive differs from the one
// they run.
func ApplyAutomaticControllerUpdates(ctx context.Context) (int64, error) {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		`UPDATE DeploymentTarget dt
		SET controller_version_id = cv.id
		FROM ControllerVersion cv
		WHERE cv.name = @name
			AND dt.automatic_updates_enabled
			AND dt.controller_version_id IS DISTINCT FROM cv.id`,
		pgx.NamedArgs{"name": buildconfig.Version()},
	)
	if err != nil {
		return 0, err
	}
	return cmd.RowsAffected(), nil
}

func GetControllerVersions(ctx context.Context) ([]types.ControllerVersion, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT cv.id, cv.created_at, cv.name, cv.manifest_file_revision, cv.compose_file_revision
		FROM ControllerVersion cv
		ORDER BY cv.created_at`,
	)
	if err != nil {
		return nil, err
	} else {
		return pgx.CollectRows(rows, pgx.RowToStructByName[types.ControllerVersion])
	}
}

func GetCurrentControllerVersion(ctx context.Context) (*types.ControllerVersion, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT cv.id, cv.created_at, cv.name, cv.manifest_file_revision, cv.compose_file_revision
		FROM ControllerVersion cv
		WHERE cv.name = @name`,
		pgx.NamedArgs{"name": buildconfig.Version()},
	)
	if err != nil {
		return nil, err
	} else if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ControllerVersion]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		} else {
			return nil, err
		}
	} else {
		return &result, nil
	}
}

func GetControllerVersionForDeploymentTargetID(ctx context.Context, id uuid.UUID) (*types.ControllerVersion, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT cv.id, cv.created_at, cv.name, cv.manifest_file_revision, cv.compose_file_revision
		FROM DeploymentTarget dt
		INNER JOIN ControllerVersion cv ON dt.controller_version_id = cv.id
		WHERE dt.id = @id`,
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, err
	} else if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ControllerVersion]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		} else {
			return nil, err
		}
	} else {
		return &result, nil
	}
}

func GetControllerVersionWithName(ctx context.Context, name string) (*types.ControllerVersion, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT cv.id, cv.created_at, cv.name, cv.manifest_file_revision, cv.compose_file_revision
		FROM ControllerVersion cv
		WHERE cv.name = @name`,
		pgx.NamedArgs{"name": name},
	)
	if err != nil {
		return nil, err
	} else if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ControllerVersion]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		} else {
			return nil, err
		}
	} else {
		return &result, nil
	}
}
