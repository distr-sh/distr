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

func CreateAgentVersion(ctx context.Context) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(ctx,
		`INSERT INTO AgentVersion (name, manifest_file_revision, compose_file_revision)
			VALUES (@name, @manifestRevision, @composeRevision)
		ON CONFLICT (name) DO UPDATE SET
			manifest_file_revision = @manifestRevision,
			compose_file_revision = @composeRevision`,
		pgx.NamedArgs{"name": buildconfig.Version(), "manifestRevision": "v2", "composeRevision": "v1"})
	return err
}

func GetAgentVersions(ctx context.Context) ([]types.AgentVersion, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT av.id, av.created_at, av.name, av.manifest_file_revision, av.compose_file_revision
		FROM AgentVersion av
		ORDER BY av.created_at`,
	)
	if err != nil {
		return nil, err
	} else {
		return pgx.CollectRows(rows, pgx.RowToStructByName[types.AgentVersion])
	}
}

func GetCurrentAgentVersion(ctx context.Context) (*types.AgentVersion, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT av.id, av.created_at, av.name, av.manifest_file_revision, av.compose_file_revision
		FROM AgentVersion av
		WHERE av.name = @name`,
		pgx.NamedArgs{"name": buildconfig.Version()},
	)
	if err != nil {
		return nil, err
	} else if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.AgentVersion]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		} else {
			return nil, err
		}
	} else {
		return &result, nil
	}
}

func GetAgentVersionForDeploymentTargetID(ctx context.Context, id uuid.UUID) (*types.AgentVersion, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT av.id, av.created_at, av.name, av.manifest_file_revision, av.compose_file_revision
		FROM DeploymentTarget dt
		INNER JOIN AgentVersion av ON dt.agent_version_id = av.id
		WHERE dt.id = @id`,
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, err
	} else if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.AgentVersion]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		} else {
			return nil, err
		}
	} else {
		return &result, nil
	}
}

func GetAgentVersionWithName(ctx context.Context, name string) (*types.AgentVersion, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT av.id, av.created_at, av.name, av.manifest_file_revision, av.compose_file_revision
		FROM AgentVersion av
		WHERE av.name = @name`,
		pgx.NamedArgs{"name": name},
	)
	if err != nil {
		return nil, err
	} else if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.AgentVersion]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		} else {
			return nil, err
		}
	} else {
		return &result, nil
	}
}
