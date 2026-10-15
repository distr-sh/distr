package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/billing"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/webhook"
	"go.uber.org/zap"
)

func stripeWebhookHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	endpointSecret := env.StripeWebhookSecret()
	if endpointSecret == nil {
		log.Warn("stripe endpoint secret not set")
		return respond.Status(http.StatusServiceUnavailable), nil
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Warn("error reading request body", zap.Error(err))
		return respond.Status(http.StatusServiceUnavailable), nil
	}

	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), *endpointSecret)
	if err != nil {
		log.Warn("webhook signature verification failed", zap.Error(err))
		if env.StripeWebhookVersionMismatchBehavior() == env.StripeWebhookVersionMismatchBehaviorIgnore &&
			event.APIVersion != "" && !strings.HasSuffix(event.APIVersion, "."+stripe.APIMajorVersion) {
			return respond.Status(http.StatusOK), nil
		}
		return respond.Status(http.StatusBadRequest), nil
	}

	log = log.With(zap.String("stripeEventId", event.ID), zap.String("stripeEventType", string(event.Type)))
	ctx = internalctx.WithLogger(ctx, log)

	switch event.Type {
	case stripe.EventTypeCustomerSubscriptionCreated:
		if _, resp, err := handleStripeSubscription(ctx, event); resp != nil || err != nil {
			return resp, err
		} else {
			log.Info("stripe customer subscription created")
		}
	case stripe.EventTypeCustomerSubscriptionUpdated:
		if _, resp, err := handleStripeSubscription(ctx, event); resp != nil || err != nil {
			return resp, err
		} else {
			log.Info("stripe customer subscription updated")
		}
	case stripe.EventTypeCustomerSubscriptionDeleted:
		if sub, resp, err := handleStripeSubscription(ctx, event); resp != nil || err != nil {
			return resp, err
		} else {
			log.Info("stripe customer subscription deleted", zap.Any("subscription", sub))
		}
	default:
		log.Info("unhandled stripe event")
	}

	return respond.Status(http.StatusOK), nil
}

// handleStripeSubscription applies the subscription of event to its organization and returns it,
// unless it answers in place of the caller.
func handleStripeSubscription(
	ctx context.Context,
	event stripe.Event,
) (*stripe.Subscription, respond.Responder, error) {
	log := internalctx.GetLogger(ctx)

	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		log.Info("error parsing webhook JSON", zap.Error(err))
		return nil, respond.Status(http.StatusBadRequest), nil
	} else if orgID, err := uuid.Parse(sub.Metadata["organizationId"]); err != nil {
		log.Warn("subscription event with missing or invalid organizationId", zap.Error(err))
		return nil, respond.NoContent(), nil
	} else if err := applyStripeSubscription(ctx, orgID, sub); err != nil {
		return nil, nil, fmt.Errorf("failed to handle stripe subscription: %w", err)
	} else {
		return &sub, nil, nil
	}
}

func applyStripeSubscription(ctx context.Context, orgID uuid.UUID, sub stripe.Subscription) error {
	log := internalctx.GetLogger(ctx)

	return db.RunTxRR(ctx, func(ctx context.Context) error {
		org, err := db.GetOrganizationByID(ctx, orgID)
		if err != nil {
			return fmt.Errorf("failed to get organization: %w", err)
		}

		org.StripeSubscriptionID = &sub.ID
		org.StripeCustomerID = &sub.Customer.ID

		var plan types.SubscriptionPlan

		if sub.Status == stripe.SubscriptionStatusCanceled {
			plan.EndsAt = time.Now()
		} else if currentPeriodEnd, err := billing.GetCurrentPeriodEnd(sub); err != nil {
			return fmt.Errorf("failed to get current period end: %w", err)
		} else {
			plan.EndsAt = *currentPeriodEnd
		}

		if subscriptionType, err := billing.GetSubscriptionType(sub); err != nil {
			return fmt.Errorf("failed to get subscription type: %w", err)
		} else {
			plan.Type = *subscriptionType
		}

		if qty, err := billing.GetCustomerOrganizationQty(sub); err != nil {
			log.Warn("could not get customer organization quantity", zap.Error(err))
			plan.CustomerOrganizationQty = 0
		} else {
			plan.CustomerOrganizationQty = qty
		}

		if qty, err := billing.GetUserAccountQty(sub); err != nil {
			return fmt.Errorf("failed to get user account quantity: %w", err)
		} else {
			plan.UserAccountQty = qty
		}

		if subscriptionPeriod, err := billing.GetSubscriptionPeriod(sub); err != nil {
			return fmt.Errorf("failed to get subscription period: %w", err)
		} else {
			plan.Period = subscriptionPeriod
		}

		org.ApplyPlan(plan)

		log.Info("updated organization subscription",
			zap.Stringer("organizationId", org.ID),
			zap.String("subscriptionType", string(org.SubscriptionType)),
			zap.Time("subscriptionEndsAt", org.SubscriptionEndsAt),
			zap.Int64("userAccountQty", org.SubscriptionUserAccountQty.Value()),
			zap.Int64("customerOrganizationQty", org.SubscriptionCustomerOrganizationQty.Value()))

		return db.UpdateOrganization(ctx, org)
	})
}
