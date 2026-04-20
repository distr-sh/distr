package svc

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/env"
	"github.com/go-mailx/mailx"
	ses "github.com/go-mailx/mailx-ses"
	smtp "github.com/go-mailx/mailx-smtp"
)

func (r *Registry) GetMailer() *mailx.Mailer {
	return r.mailer
}

func createMailer(ctx context.Context) (*mailx.Mailer, error) {
	config := env.GetMailerConfig()
	authOrgOverrideFromAddress := func(ctx context.Context, mail mailx.Mail) string {
		if auth, err := auth.Authentication.Get(ctx); err == nil {
			if org := auth.CurrentOrg(); org != nil && org.EmailFromAddress != nil {
				return *org.EmailFromAddress
			}
		}
		return ""
	}

	var adapter mailx.MailerAdapter
	var err error

	switch config.Type {
	case env.MailerTypeSMTP:
		smtpConfig := smtp.Config{
			Host:        config.SmtpConfig.Host,
			Port:        config.SmtpConfig.Port,
			Username:    config.SmtpConfig.Username,
			Password:    config.SmtpConfig.Password,
			ImplicitTLS: config.SmtpConfig.ImplicitTLS,
			TLSPolicy:   smtp.TLSOpportunistic,
		}
		adapter, err = smtp.New(smtpConfig)
	case env.MailerTypeSES:
		adapter, err = ses.NewFromContext(ctx)
	case env.MailerTypeUnspecified:
		adapter = &mailx.Noop{}
	default:
		err = errors.New("invalid mailer type")
	}

	if err != nil {
		return nil, fmt.Errorf("mailer creation failed: %w", err)
	}

	return &mailx.Mailer{MailerAdapter: adapter, Config: &mailx.MailerConfig{
		FromAddressSrc: []mailx.FromAddressFunc{
			mailx.MailOverrideFromAddress(),
			authOrgOverrideFromAddress,
			mailx.StaticFromAddress(config.FromAddress.String()),
		},
	}}, nil
}
