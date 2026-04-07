package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/billing"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"
	"go.uber.org/zap"
)

var errNoOrganization = errors.New("stripe subscription metadata missing or invalid organizationId")

func stripeWebhookHandler() http.HandlerFunc {
	endpointSecret := env.StripeWebhookSecret()

	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		log := internalctx.GetLogger(ctx)

		if endpointSecret == nil {
			log.Warn("stripe endpoint secret not set")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		payload, err := io.ReadAll(req.Body)
		if err != nil {
			log.Warn("error reading request body", zap.Error(err))
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		event, err := webhook.ConstructEvent(payload, req.Header.Get("Stripe-Signature"), *endpointSecret)
		if err != nil {
			log.Warn("webhook signature verification failed", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log = log.With(zap.String("stripeEventId", event.ID), zap.String("stripeEventType", string(event.Type)))
		ctx = internalctx.WithLogger(ctx, log)

		switch event.Type {
		case stripe.EventTypeCustomerSubscriptionCreated:
			var subscription stripe.Subscription
			err = json.Unmarshal(event.Data.Raw, &subscription)
			if err != nil {
				log.Info("Error parsing webhook JSON", zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			log.Info("stripe customer subscription created")
			err = handleStripeSubscription(ctx, subscription)
		case stripe.EventTypeCustomerSubscriptionUpdated:
			var subscription stripe.Subscription
			err = json.Unmarshal(event.Data.Raw, &subscription)
			if err != nil {
				log.Info("Error parsing webhook JSON", zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			log.Info("stripe customer subscription updated")
			err = handleStripeSubscription(ctx, subscription)
		case stripe.EventTypeCustomerSubscriptionDeleted:
			var subscription stripe.Subscription
			err = json.Unmarshal(event.Data.Raw, &subscription)
			if err != nil {
				log.Info("Error parsing webhook JSON", zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			log.Info("stripe customer subscription deleted", zap.Any("subscription", subscription))
			err = handleStripeSubscription(ctx, subscription)
		default:
			log.Info("unhandled stripe event")
		}

		if errors.Is(err, errNoOrganization) {
			w.WriteHeader(http.StatusNoContent)
		} else if err != nil {
			log.Error("Error handling stripe subscription", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			switch {
			case errors.Is(err, apierrors.ErrBadRequest):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, apierrors.ErrNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

func handleStripeSubscription(ctx context.Context, sub stripe.Subscription) error {
	log := internalctx.GetLogger(ctx)

	orgID, err := uuid.Parse(sub.Metadata["organizationId"])
	if err != nil {
		log.Warn("subscription event with missing or invalid organizationId", zap.Error(err))
		return fmt.Errorf("%w: %w", errNoOrganization, err)
	}

	return db.RunTxRR(ctx, func(ctx context.Context) error {
		org, err := db.GetOrganizationByID(ctx, orgID)
		if err != nil {
			return err
		}

		org.StripeSubscriptionID = &sub.ID
		org.StripeCustomerID = &sub.Customer.ID

		if sub.Status == stripe.SubscriptionStatusCanceled {
			org.SubscriptionEndsAt = time.Now()
		} else if currentPeriodEnd, err := billing.GetCurrentPeriodEnd(sub); err != nil {
			return fmt.Errorf("%w: %w", apierrors.ErrBadRequest, err)
		} else {
			org.SubscriptionEndsAt = *currentPeriodEnd
		}

		if subscriptionType, err := billing.GetSubscriptionType(sub); err != nil {
			return fmt.Errorf("%w: %w", apierrors.ErrBadRequest, err)
		} else {
			org.SubscriptionType = *subscriptionType
		}

		if qty, err := billing.GetCustomerOrganizationQty(sub); err != nil {
			log.Warn("could not get customer organization quantity", zap.Error(err))
			org.SubscriptionCustomerOrganizationQty = 0
		} else {
			org.SubscriptionCustomerOrganizationQty = qty
		}

		if qty, err := billing.GetUserAccountQty(sub); err != nil {
			return fmt.Errorf("%w: %w", apierrors.ErrBadRequest, err)
		} else {
			org.SubscriptionUserAccountQty = qty
		}

		if subscriptionPeriod, err := billing.GetSubscriptionPeriod(sub); err != nil {
			return fmt.Errorf("%w: %w", apierrors.ErrBadRequest, err)
		} else {
			org.SubscriptionPeriod = subscriptionPeriod
		}

		if org.SubscriptionType == types.SubscriptionTypeStarter {
			org.RemoveFeatures(subscription.ProFeatures...)
		} else {
			org.AddFeatures(subscription.ProFeatures...)
		}

		log.Info("updated organization subscription",
			zap.Stringer("organizationId", org.ID),
			zap.String("subscriptionType", string(org.SubscriptionType)),
			zap.Time("subscriptionEndsAt", org.SubscriptionEndsAt),
			zap.Int64("userAccountQty", org.SubscriptionUserAccountQty.Value()),
			zap.Int64("customerOrganizationQty", org.SubscriptionCustomerOrganizationQty.Value()))

		if err := db.UpdateOrganization(ctx, org); err != nil {
			return err
		}

		if org.SubscriptionType == types.SubscriptionTypeStarter {
			if err := subscription.ReconcileStarterFeaturesForOrganizationID(ctx, orgID); err != nil {
				return err
			}
		}

		return nil
	})
}
