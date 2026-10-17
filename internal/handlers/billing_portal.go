package handlers

import (
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/billing"
	"github.com/distr-sh/distr/internal/handlerutil"
	"github.com/distr-sh/distr/internal/respond"
)

func CreateBillingPortalSessionHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	org := auth.CurrentOrg()

	// Check if organization has a Stripe customer ID
	if org.StripeCustomerID == nil || *org.StripeCustomerID == "" {
		return respond.Conflict("no stripe customer found for organization"), nil
	} else if _, resp := decodeJSON[struct {
		ReturnURL string `json:"returnUrl"`
	}](r); resp != nil {
		return resp, nil
	} else if session, err := billing.CreateBillingPortalSession(ctx, billing.BillingPortalSessionParams{
		CustomerID: *org.StripeCustomerID,
		ReturnURL:  fmt.Sprintf("%v/subscription", handlerutil.GetRequestSchemeAndHost(r)),
	}); err != nil {
		return nil, fmt.Errorf("failed to create billing portal session: %w", err)
	} else {
		return respond.JSON(map[string]string{
			"url": session.URL,
		}), nil
	}
}
