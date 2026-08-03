// Package custommail resolves the mail transport and sender address to use for mails sent on
// behalf of an organization. An organization may configure its own SMTP server, which overrides
// the instance mailer built from the MAILER_* environment variables.
package custommail

import (
	"context"
	"errors"
	"fmt"
	"net/mail"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/go-mailx/mailx"
	smtp "github.com/go-mailx/mailx-smtp"
	"github.com/google/uuid"
)

// MailerForOrganization returns the mailer to send mails on behalf of the given organization
// with: the organization's own configuration if it has an enabled one, the instance mailer
// otherwise.
//
// There is deliberately no fallback to the instance mailer when the organization's configuration
// cannot be resolved or used: the instance mailer sends from a domain the organization's sender
// address does not belong to, which fails SPF/DKIM/DMARC and hides the misconfiguration.
func MailerForOrganization(ctx context.Context, orgID uuid.UUID) (*mailx.Mailer, error) {
	config, err := configuration(ctx, orgID)
	if err != nil {
		return nil, err
	} else if config == nil {
		return internalctx.GetMailer(ctx), nil
	}
	return MailerForConfiguration(*config)
}

// MailerForConfiguration builds a mailer for the given configuration without consulting the
// database, so that a configuration can be tested before it is stored.
func MailerForConfiguration(config types.CustomEmailConfiguration) (*mailx.Mailer, error) {
	adapter, err := smtp.New(smtp.Config{
		Host:        config.SMTPHost,
		Port:        config.SMTPPort,
		Username:    config.SMTPUsername,
		Password:    config.SMTPPassword,
		ImplicitTLS: config.SMTPImplicitTLS,
		TLSPolicy:   smtp.TLSOpportunistic,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create mailer for custom email configuration: %w", err)
	}
	return &mailx.Mailer{MailerAdapter: adapter, Config: &mailx.MailerConfig{
		FromAddressSrc: []mailx.FromAddressFunc{
			mailx.MailOverrideFromAddress(),
			mailx.StaticFromAddress(config.FromAddress),
		},
	}}, nil
}

// FromAddressOrDefault resolves the sender address for an organization: its custom email
// configuration first, then the legacy OrganizationBranding.email_from_address, then the
// instance default from MAILER_FROM_ADDRESS.
func FromAddressOrDefault(
	ctx context.Context,
	orgID uuid.UUID,
	b *types.OrganizationBranding,
) (*mail.Address, error) {
	config, err := configuration(ctx, orgID)
	if err != nil {
		return nil, err
	} else if config != nil {
		return mail.ParseAddress(config.FromAddress)
	}
	if b != nil && b.EmailFromAddress != nil {
		return mail.ParseAddress(*b.EmailFromAddress)
	}
	return new(env.GetMailerConfig().FromAddress), nil
}

// configuration returns the organization's email configuration if it has an enabled one, and nil
// if it has none or a disabled one.
func configuration(ctx context.Context, orgID uuid.UUID) (*types.CustomEmailConfiguration, error) {
	config, err := db.GetCustomEmailConfiguration(ctx, orgID)
	if errors.Is(err, apierrors.ErrNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else if !config.Enabled {
		return nil, nil
	}
	return config, nil
}
