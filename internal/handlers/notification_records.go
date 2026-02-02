package handlers

import (
	"net/http"

	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/getsentry/sentry-go"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func NotificationRecordsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Notifications"))

	r.Get("/", getNotificationRecordsHandler()).With()
}

func getNotificationRecordsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		records, err := db.GetNotificationRecords(ctx, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID())
		if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get notification records", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, records)
	}
}
