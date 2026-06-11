package db

import (
	"context"
	"errors"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TryClaimNotificationEmailQuota atomically inserts a NotificationEmailRecord for the given
// email address if fewer than quota records exist for it within the last hour. It returns the
// ID of the inserted record, or nil if the quota is exhausted. Records older than one hour are
// deleted opportunistically.
func TryClaimNotificationEmailQuota(ctx context.Context, email string, quota int) (*uuid.UUID, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`WITH cleanup AS (
			DELETE FROM NotificationEmailRecord WHERE created_at < now() - interval '1 hour'
		), inserted AS (
			INSERT INTO NotificationEmailRecord (email)
			SELECT @email
			WHERE (
				SELECT count(*) FROM NotificationEmailRecord
				WHERE email = @email AND created_at > now() - interval '1 hour'
			) < @quota
			RETURNING id
		)
		SELECT id FROM inserted`,
		pgx.NamedArgs{
			"email": email,
			"quota": quota,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to claim notification email quota: %w", err)
	}

	if id, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[uuid.UUID]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to collect NotificationEmailRecord ID: %w", err)
	} else {
		return &id, nil
	}
}

// DeleteNotificationEmailRecord releases a previously claimed quota slot,
// e.g. when the email could not be sent.
func DeleteNotificationEmailRecord(ctx context.Context, id uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	if _, err := db.Exec(
		ctx,
		`DELETE FROM NotificationEmailRecord WHERE id = @id`,
		pgx.NamedArgs{"id": id},
	); err != nil {
		return fmt.Errorf("failed to delete NotificationEmailRecord: %w", err)
	}
	return nil
}
