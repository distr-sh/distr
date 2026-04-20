package mailsending

import (
	"context"
	"encoding/json"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/types"
	"github.com/go-mailx/mailx"
	"go.uber.org/zap"
)

func SendLicenseKeyRevisedCustomer(
	ctx context.Context,
	user types.UserAccount,
	licenseKey types.LicenseKey,
	revision types.LicenseKeyRevision,
	token string,
) error {
	mailer := internalctx.GetMailer(ctx)
	log := internalctx.GetLogger(ctx)

	payloadFormatted, err := formatPayload(revision.Payload)
	if err != nil {
		return fmt.Errorf("could not format license key payload: %w", err)
	}

	if err := mailer.Send(ctx,
		mailx.To(user.Email),
		mailx.Subject(fmt.Sprintf("License key updated: %s", licenseKey.Name)),
		mailx.HtmlBodyTemplate(mailtemplates.LicenseKeyRevisedCustomer(licenseKey, revision, payloadFormatted, token)),
	); err != nil {
		log.Error("could not send license key revised mail to customer user",
			zap.Error(err),
			zap.String("email", user.Email),
		)
		return err
	}
	return nil
}

func SendLicenseKeyRevisedVendor(
	ctx context.Context,
	user types.UserAccount,
	licenseKey types.LicenseKey,
	revision types.LicenseKeyRevision,
	customerOrgName string,
) error {
	mailer := internalctx.GetMailer(ctx)
	log := internalctx.GetLogger(ctx)

	payloadFormatted, err := formatPayload(revision.Payload)
	if err != nil {
		return fmt.Errorf("could not format license key payload: %w", err)
	}

	subject := fmt.Sprintf("License key updated: %s", licenseKey.Name)
	if customerOrgName != "" {
		subject = fmt.Sprintf("License key updated for %s: %s", customerOrgName, licenseKey.Name)
	}

	if err := mailer.Send(ctx,
		mailx.To(user.Email),
		mailx.Subject(subject),
		mailx.HtmlBodyTemplate(
			mailtemplates.LicenseKeyRevisedVendor(licenseKey, revision, payloadFormatted, customerOrgName),
		),
	); err != nil {
		log.Error("could not send license key revised mail to vendor user",
			zap.Error(err),
			zap.String("email", user.Email),
		)
		return err
	}
	return nil
}

func formatPayload(payload json.RawMessage) (string, error) {
	var raw any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return "", err
	}
	formatted, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}
