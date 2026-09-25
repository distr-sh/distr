package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var customEmailConfigurationOutputExpr = `
	c.id, c.created_at, c.updated_at, c.updated_by_user_account_id, c.organization_id, c.enabled,
	c.from_address, c.smtp_host, c.smtp_port, ` +
	emailSMTPUsername.Output("c") + `, ` +
	emailSMTPPassword.Output("c") + `, c.smtp_implicit_tls
`

// The result includes the SMTP password, which must never be returned to a client.
func GetCustomEmailConfiguration(
	ctx context.Context,
	organizationID uuid.UUID,
) (*types.CustomEmailConfiguration, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT"+customEmailConfigurationOutputExpr+
			"FROM CustomEmailConfiguration c WHERE c.organization_id = @organizationId",
		pgx.NamedArgs{"organizationId": organizationID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query CustomEmailConfiguration: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomEmailConfiguration])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("could not collect CustomEmailConfiguration: %w", err)
	}
	return &result, nil
}

// The stored state is written back into the given struct.
func UpsertCustomEmailConfiguration(ctx context.Context, config *types.CustomEmailConfiguration) error {
	// Both values are bound to the organization rather than to the row, because the conflict path of
	// the upsert below keeps the id of the row it finds instead of one generated here.
	smtpUsernameEnc, err := emailSMTPUsername.Encrypt(config.SMTPUsername, config.OrganizationID)
	if err != nil {
		return fmt.Errorf("could not encrypt SMTP username: %w", err)
	}
	smtpPasswordEnc, err := emailSMTPPassword.Encrypt(config.SMTPPassword, config.OrganizationID)
	if err != nil {
		return fmt.Errorf("could not encrypt SMTP password: %w", err)
	}
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`INSERT INTO CustomEmailConfiguration AS c (
			updated_by_user_account_id, organization_id, enabled, from_address,
			smtp_host, smtp_port, smtp_username_enc, smtp_password_enc, smtp_implicit_tls
		) VALUES (
			@updatedByUserAccountId, @organizationId, @enabled, @fromAddress,
			@smtpHost, @smtpPort, @smtpUsernameEnc, @smtpPasswordEnc, @smtpImplicitTls
		) ON CONFLICT (organization_id) DO UPDATE SET
			updated_at = now(),
			updated_by_user_account_id = excluded.updated_by_user_account_id,
			enabled = excluded.enabled,
			from_address = excluded.from_address,
			smtp_host = excluded.smtp_host,
			smtp_port = excluded.smtp_port,
			smtp_username = NULL,
			smtp_username_enc = excluded.smtp_username_enc,
			smtp_password = NULL,
			smtp_password_enc = excluded.smtp_password_enc,
			smtp_implicit_tls = excluded.smtp_implicit_tls
		RETURNING`+customEmailConfigurationOutputExpr,
		pgx.NamedArgs{
			"updatedByUserAccountId": config.UpdatedByUserAccountID,
			"organizationId":         config.OrganizationID,
			"enabled":                config.Enabled,
			"fromAddress":            config.FromAddress,
			"smtpHost":               config.SMTPHost,
			"smtpPort":               config.SMTPPort,
			"smtpUsernameEnc":        smtpUsernameEnc,
			"smtpPasswordEnc":        smtpPasswordEnc,
			"smtpImplicitTls":        config.SMTPImplicitTLS,
		},
	)
	if err != nil {
		return fmt.Errorf("could not upsert CustomEmailConfiguration: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomEmailConfiguration])
	if err != nil {
		return fmt.Errorf("could not collect CustomEmailConfiguration: %w", err)
	}
	*config = result
	return nil
}

func DeleteCustomEmailConfiguration(ctx context.Context, organizationID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		"DELETE FROM CustomEmailConfiguration WHERE organization_id = @organizationId",
		pgx.NamedArgs{"organizationId": organizationID},
	)
	if err != nil {
		return fmt.Errorf("could not delete CustomEmailConfiguration: %w", err)
	} else if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}
