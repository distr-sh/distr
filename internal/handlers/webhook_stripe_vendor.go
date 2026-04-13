package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/billing"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/licensekey"
	"github.com/distr-sh/distr/internal/licensetemplate"
	"github.com/distr-sh/distr/internal/mailsending"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/webhook"
	"go.uber.org/zap"
)

func vendorStripeWebhookHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		log := internalctx.GetLogger(ctx)

		orgID, err := uuid.Parse(req.PathValue("organizationId"))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		org, err := db.GetOrganizationByID(ctx, orgID)
		if errors.Is(err, apierrors.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if err != nil {
			log.Error("failed to get organization for vendor webhook", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if org.StripeWebhookSecret == nil {
			log.Warn("vendor stripe webhook secret not configured", zap.Stringer("organizationId", orgID))
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		payload, err := io.ReadAll(req.Body)
		if err != nil {
			log.Warn("error reading vendor webhook request body", zap.Error(err))
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		event, err := webhook.ConstructEvent(payload, req.Header.Get("Stripe-Signature"), *org.StripeWebhookSecret)
		if err != nil {
			log.Warn("vendor webhook signature verification failed", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log = log.With(
			zap.Stringer("organizationId", orgID),
			zap.String("stripeEventId", event.ID),
			zap.String("stripeEventType", string(event.Type)),
		)
		ctx = internalctx.WithLogger(ctx, log)

		switch event.Type {
		case stripe.EventTypeCustomerSubscriptionCreated,
			stripe.EventTypeCustomerSubscriptionUpdated,
			stripe.EventTypeCustomerSubscriptionDeleted:
			var sub stripe.Subscription
			if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
				log.Info("error parsing vendor webhook JSON", zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if err := handleVendorStripeSubscription(ctx, orgID, sub); err != nil {
				log.Error("error handling vendor stripe subscription", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		default:
			log.Info("unhandled vendor stripe event")
		}

		w.WriteHeader(http.StatusOK)
	}
}

func handleVendorStripeSubscription(ctx context.Context, orgID uuid.UUID, sub stripe.Subscription) error {
	log := internalctx.GetLogger(ctx)

	licenseKeyID, err := uuid.Parse(sub.Metadata["licenseKeyId"])
	if err != nil {
		log.Info("vendor subscription event with missing or invalid licenseKeyId", zap.Error(err))
		return nil
	}

	licenseKey, err := db.GetLicenseKeyByID(ctx, licenseKeyID)
	if errors.Is(err, apierrors.ErrNotFound) {
		log.Info("license key not found for vendor subscription event", zap.Stringer("licenseKeyId", licenseKeyID))
		return nil
	} else if err != nil {
		return err
	}

	if licenseKey.OrganizationID != orgID {
		log.Info("license key does not belong to organization",
			zap.Stringer("licenseKeyId", licenseKeyID),
			zap.Stringer("organizationId", orgID),
		)
		return nil
	}

	if licenseKey.LicenseTemplateID == nil {
		log.Info("license key has no template, skipping vendor subscription event",
			zap.Stringer("licenseKeyId", licenseKeyID),
		)
		return nil
	}

	tmpl, err := db.GetLicenseTemplateByID(ctx, *licenseKey.LicenseTemplateID, orgID)
	if err != nil {
		return err
	}

	payload, err := licensetemplate.RenderPayload(*tmpl, sub)
	if err != nil {
		return err
	}

	eventTime := time.Now()
	if sub.Status == stripe.SubscriptionStatusCanceled && sub.CanceledAt != 0 {
		eventTime = time.Unix(sub.CanceledAt, 0)
	} else if periodEnd, err := billing.GetCurrentPeriodEnd(sub); err == nil {
		eventTime = *periodEnd
	}
	notBefore := time.Now()
	expiresAt := eventTime.AddDate(0, 0, tmpl.ExpirationGracePeriodDays)
	if notBefore.After(expiresAt) {
		notBefore = expiresAt
	}

	if licenseKey.NotBefore != nil && licenseKey.ExpiresAt != nil && licenseKey.Payload != nil {
		payloadEqual, err := payloadsEqual(licenseKey.Payload, payload)
		if err != nil {
			return err
		}
		if payloadEqual &&
			licenseKey.NotBefore.Equal(notBefore.UTC().Truncate(time.Second)) &&
			licenseKey.ExpiresAt.Equal(expiresAt.UTC().Truncate(time.Second)) {
			log.Info("license key revision unchanged, skipping", zap.Stringer("licenseKeyId", licenseKeyID))
			return nil
		}
	}

	revision := types.LicenseKeyRevision{
		LicenseKeyID: licenseKeyID,
		NotBefore:    notBefore,
		ExpiresAt:    expiresAt,
		Payload:      payload,
	}

	log.Info("creating license key revision from vendor stripe subscription",
		zap.Stringer("licenseKeyId", licenseKeyID),
		zap.Time("expiresAt", expiresAt),
	)

	if err := db.CreateLicenseKeyRevision(ctx, &revision); err != nil {
		return err
	}

	sendLicenseKeyRevisionEmails(ctx, licenseKey, revision)
	return nil
}

func sendLicenseKeyRevisionEmails(
	ctx context.Context,
	licenseKey *types.LicenseKey,
	revision types.LicenseKeyRevision,
) {
	log := internalctx.GetLogger(ctx)

	vendorUsers, err := db.GetUserAccountsByOrgID(ctx, licenseKey.OrganizationID)
	if err != nil {
		log.Error("failed to get vendor users for license key revision notification", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		return
	}

	var customerOrgName string
	if licenseKey.CustomerOrganizationID != nil {
		customerOrg, err := db.GetCustomerOrganizationByID(ctx, *licenseKey.CustomerOrganizationID)
		if err != nil {
			log.Error("failed to get customer organization for license key revision notification", zap.Error(err))
		} else {
			customerOrgName = customerOrg.Name
		}
	}

	token, err := licensekey.GenerateToken(licensekey.FromLicenseKeyAndRevision(*licenseKey, revision), env.Host())
	if err != nil {
		log.Error("failed to generate license key token for revision notification", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		return
	}

	for _, u := range vendorUsers {
		if u.UserRole != types.UserRoleAdmin || !u.EmailVerified {
			continue
		}

		var err error
		if u.CustomerOrganizationID == nil {
			err = mailsending.SendLicenseKeyRevisedVendor(ctx, u.AsUserAccount(), *licenseKey, revision, customerOrgName)
		} else if util.PtrEq(u.CustomerOrganizationID, licenseKey.CustomerOrganizationID) {
			err = mailsending.SendLicenseKeyRevisedCustomer(ctx, u.AsUserAccount(), *licenseKey, revision, token)
		}

		if err != nil {
			log.Error("failed to send license key revised mail", zap.Error(err), zap.String("email", u.Email))
		}
	}
}
