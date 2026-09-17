package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	fileOutputExpr = "f.id, f.organization_id, f.created_at, f.content_type, f.data, f.file_name, f.file_size, f.public"
)

func CreateFile(ctx context.Context, organizationID *uuid.UUID, file *types.File) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"INSERT INTO File AS f (organization_id, content_type, data, file_name, file_size, public) "+
			"VALUES (@organization_id, @content_type, @data, @file_name, @file_size, @public) "+
			"RETURNING "+fileOutputExpr,
		pgx.NamedArgs{
			"organization_id": organizationID,
			"content_type":    file.ContentType,
			"data":            file.Data,
			"file_name":       file.FileName,
			"file_size":       file.FileSize,
			"public":          file.Public,
		},
	)
	if err != nil {
		return fmt.Errorf("could not query file: %w", err)
	} else if created, err := pgx.CollectExactlyOneRow[types.File](rows, pgx.RowToStructByName); err != nil {
		return fmt.Errorf("could not create file: %w", err)
	} else {
		*file = created
		return nil
	}
}

func GetFileWithID(ctx context.Context, id uuid.UUID) (*types.File, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT "+fileOutputExpr+" FROM File f WHERE f.id = @id",
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query file: %w", err)
	} else if file, err := pgx.CollectExactlyOneRow[types.File](rows, pgx.RowToStructByName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		} else {
			return nil, fmt.Errorf("could not map file: %w", err)
		}
	} else {
		return &file, nil
	}
}

// GetFileMetadataWithID loads only the ownership and visibility of a file, avoiding the data blob.
func GetFileMetadataWithID(ctx context.Context, id uuid.UUID) (*types.FileMetadata, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT f.organization_id, f.content_type, f.public FROM File f WHERE f.id = @id",
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query file metadata: %w", err)
	} else if metadata, err := pgx.CollectExactlyOneRow[types.FileMetadata](rows, pgx.RowToStructByName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		} else {
			return nil, fmt.Errorf("could not map file metadata: %w", err)
		}
	} else {
		return &metadata, nil
	}
}

func DeleteFileWithID(ctx context.Context, id uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx, `DELETE FROM file WHERE id = @id`, pgx.NamedArgs{"id": id})
	if err != nil {
		if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.ForeignKeyViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
	} else if cmd.RowsAffected() == 0 {
		err = apierrors.ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("could not delete File: %w", err)
	}

	return nil
}

// DeleteUnreferencedFilesOlderThan collects every image that was replaced, since attaching a new one
// only overwrites the id and leaves the previous file behind, plus the avatars of the accounts
// DeleteUserAccountsOlderThan removed.
//
// minAge has to outlast the flows that upload an image before anything references it: the organization
// branding form and the user profile settings stage the returned id and attach it only when the form
// is saved, so a young file may still be waiting for a save. Everywhere else attaches one request
// later. Deleting such a file makes that save fail with "file does not exist".
func DeleteUnreferencedFilesOlderThan(ctx context.Context, minAge time.Duration) (int64, error) {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(
		ctx,
		`DELETE FROM File AS f
		WHERE now() - f.created_at > @minAge
			AND NOT EXISTS (SELECT 1 FROM UserAccount u WHERE u.image_id = f.id)
			AND NOT EXISTS (SELECT 1 FROM Application a WHERE a.image_id = f.id)
			AND NOT EXISTS (SELECT 1 FROM Artifact a WHERE a.image_id = f.id)
			AND NOT EXISTS (SELECT 1 FROM CustomerOrganization c WHERE c.image_id = f.id)
			AND NOT EXISTS (
				SELECT 1 FROM OrganizationBranding b
				WHERE b.logo_image_id = f.id OR b.favicon_image_id = f.id
			)`,
		pgx.NamedArgs{"minAge": minAge},
	)
	if err != nil {
		return 0, fmt.Errorf("could not delete unreferenced files: %w", err)
	}
	return cmd.RowsAffected(), nil
}
