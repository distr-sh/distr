package billing

import (
	"context"

	"github.com/distr-sh/distr/internal/util"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/billingportal/session"
)

type BillingPortalSessionParams struct {
	CustomerID string
	ReturnURL  string
}

func CreateBillingPortalSession(
	ctx context.Context,
	params BillingPortalSessionParams,
) (*stripe.BillingPortalSession, error) {
	sessionParams := &stripe.BillingPortalSessionParams{
		Params:    stripe.Params{Context: ctx},
		Customer:  util.PtrTo(params.CustomerID),
		ReturnURL: util.PtrTo(params.ReturnURL),
	}

	return session.New(sessionParams)
}
