package handlers

import (
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/getsentry/sentry-go"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func NotificationRecordsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Notifications"))

	r.Use(middleware.ProFeature)

	r.Get("/", getNotificationRecordsHandler()).
		With(option.Request(customerScopeQuery{})).
		With(option.Response(http.StatusOK, []api.NotificationRecord{}))
}

func getNotificationRecordsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		customerOrgID, ok := resolveCustomerScopeFromQuery(w, r)
		if !ok {
			return
		}

		records, err := db.GetNotificationRecords(ctx, *auth.CurrentOrgID(), customerOrgID)
		if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get notification records", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.List(records, mapping.NotificationRecordToAPI))
	}
}
