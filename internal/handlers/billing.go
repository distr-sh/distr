package handlers

import (
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func BillingRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.Use(middleware.RequireOrgAndRole, middleware.RequireVendor)
	r.Route("/subscription", func(r chiopenapi.Router) {
		r.Get("/", GetSubscriptionHandler)
		r.Group(func(r chiopenapi.Router) {
			r.Use(middleware.RequireAdmin, middleware.BlockSuperAdmin)
			r.Post("/", CreateSubscriptionHandler)
			r.Put("/", UpdateSubscriptionHandler)
		})
	})
	r.Group(func(r chiopenapi.Router) {
		r.Use(middleware.RequireAdmin, middleware.BlockSuperAdmin)
		r.Post("/portal", CreateBillingPortalSessionHandler)
	})
}
