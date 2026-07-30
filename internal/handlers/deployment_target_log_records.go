package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/limit"
	"github.com/distr-sh/distr/internal/logstore"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

func getDeploymentTargetLogRecordsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		deploymentTarget := internalctx.GetDeploymentTarget(ctx)

		limitParam, err := parseTimeseriesLimit(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		order := types.OrderDirection(r.FormValue("order"))

		authInfo := auth.Authentication.Require(ctx)
		org := authInfo.CurrentOrg()
		queryRange, err := parseLogQueryRange(r, org.SubscriptionType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		direction := types.EffectiveOrderDirection(order, queryRange.StartExplicit)
		if queryRange.IsEmpty() {
			RespondJSON(w, mapping.List(nil, mapping.DeploymentTargetLogRecordToAPI))
			return
		}

		logStore := logstore.FromContext(ctx)
		records, err := util.SeqCollect(logStore.QueryDeploymentTargetLogRecords(ctx, org.ID,
			logstore.DeploymentTargetLogQuery{
				DeploymentTargetID: deploymentTarget.ID,
				Start:              queryRange.Start,
				End:                queryRange.End,
				Filter:             queryRange.Filter,
				Limit:              limit.Limit(limitParam),
				Direction:          direction,
			}))
		if err != nil {
			if errors.Is(err, apierrors.ErrBadRequest) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			internalctx.GetLogger(ctx).Error("failed to get deployment target log records", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.List(records, mapping.DeploymentTargetLogRecordToAPI))
	}
}

func exportDeploymentTargetLogRecordsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		deploymentTarget := internalctx.GetDeploymentTarget(ctx)
		authInfo := auth.Authentication.Require(ctx)
		org := authInfo.CurrentOrg()

		queryRange, err := parseLogQueryRange(r, org.SubscriptionType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		filename := fmt.Sprintf("%s_agent.log", time.Now().Format("2006-01-02"))
		export := newExportWriter(w, log, filename)

		if queryRange.IsEmpty() {
			export.finish()
			return
		}

		logStore := logstore.FromContext(ctx)
		records := logStore.QueryDeploymentTargetLogRecords(ctx, org.ID, logstore.DeploymentTargetLogQuery{
			DeploymentTargetID: deploymentTarget.ID,
			Start:              queryRange.Start,
			End:                queryRange.End,
			Filter:             queryRange.Filter,
			Limit:              subscription.MaxLogExportRows,
			Direction:          types.OrderDirectionDesc,
		})
		for record, err := range records {
			if err != nil {
				export.fail(ctx, "failed to export deployment target log records", err)
				return
			}
			if err := export.writeLine("%s\t%s\t%s\n",
				record.Timestamp.Format(time.RFC3339),
				record.Severity,
				strings.TrimSpace(record.Body)); err != nil {
				return
			}
		}
		export.finish()
	}
}
