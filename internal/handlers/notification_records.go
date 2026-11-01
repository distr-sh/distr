package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

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
		With(option.Request(struct {
			customerScopeQuery
			After *time.Time `query:"after"`
		}{})).
		With(option.Response(http.StatusOK, []api.NotificationRecord{}))
}

func getNotificationRecordsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	var after *time.Time
	if value, err := QueryParam(r, "after", ParseTimeFunc(time.RFC3339Nano)); err == nil {
		after = &value
	} else if !errors.Is(err, ErrParamNotDefined) {
		return respond.BadRequest("after must be a valid date"), nil
	}

	if customerOrgID, resp, err := customerScopeFromQuery(r); resp != nil || err != nil {
		return resp, err
	} else if records, err := db.GetNotificationRecords(ctx, *auth.CurrentOrgID(), customerOrgID, after); err != nil {
		return nil, fmt.Errorf("failed to get notification records: %w", err)
	} else {
		return respond.JSON(mapping.List(records, mapping.NotificationRecordWithDetailsToAPI)), nil
	}
}
