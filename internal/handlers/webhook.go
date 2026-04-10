package handlers

import (
	"github.com/go-chi/chi/v5/middleware"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func WebhookRouter(r chiopenapi.Router) {
	const MaxBodyBytes = int64(65536)

	r.With(middleware.RequestSize(MaxBodyBytes)).Group(func(r chiopenapi.Router) {
		r.Post("/stripe", stripeWebhookHandler()).With(option.Hidden(true))
		r.Post("/{organizationId}/stripe", vendorStripeWebhookHandler()).With(option.Hidden(true))
	})
}
