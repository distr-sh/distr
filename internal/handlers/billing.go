package handlers

import (
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func BillingRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.Use(middleware.RequireOrgAndRole, middleware.RequireVendor)
	r.Route("/subscription", func(r chiopenapi.Router) {
		r.Get("/", respond.Handler(GetSubscriptionHandler))
		r.Group(func(r chiopenapi.Router) {
			r.Use(middleware.RequireAdmin, middleware.BlockSuperAdmin)
			r.Post("/", respond.Handler(CreateSubscriptionHandler))
			r.Put("/", respond.Handler(UpdateSubscriptionHandler))
		})
	})
	r.Group(func(r chiopenapi.Router) {
		r.Use(middleware.RequireAdmin, middleware.BlockSuperAdmin)
		r.Post("/portal", respond.Handler(CreateBillingPortalSessionHandler))
	})
}
