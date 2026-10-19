package handlers

import (
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func NotificationRecordsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Notifications"))

	r.Use(middleware.ProFeature, middleware.NotificationHistoryCustomerFeatureMiddleware)

	r.Get("/", respond.Handler(getNotificationRecordsHandler)).
		With(option.Request(customerScopeQuery{})).
		With(option.Response(http.StatusOK, []api.NotificationRecord{}))
}

func getNotificationRecordsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if customerOrgID, resp, err := customerScopeFromQuery(r); resp != nil || err != nil {
		return resp, err
	} else if records, err := db.GetNotificationRecords(ctx, *auth.CurrentOrgID(), customerOrgID); err != nil {
		return nil, fmt.Errorf("failed to get notification records: %w", err)
	} else {
		return respond.JSON(mapping.List(records, mapping.NotificationRecordWithDetailsToAPI)), nil
	}
}
