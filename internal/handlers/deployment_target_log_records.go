package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
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

		limitParam, err := QueryParam(r, "limit", strconv.Atoi, Min(1), Max(100))
		if errors.Is(err, ErrParamNotDefined) {
			limitParam = 25
		} else if err != nil {
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

		if queryRange.IsEmpty() {
			SetFileDownloadHeaders(w, filename)
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

		// The download headers are only set right before the first write, so an error
		// response can still be sent as long as nothing has been written yet.
		written := false
		count := int64(0)
		for record, err := range records {
			if err != nil {
				log.Error("failed to export deployment target log records", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				if !written {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				} else {
					_, _ = w.Write([]byte(exportTruncationNotice))
				}
				return
			}
			if !written {
				SetFileDownloadHeaders(w, filename)
				written = true
			}
			_, err := fmt.Fprintf(w, "%s\t%s\t%s\n",
				record.Timestamp.Format(time.RFC3339), record.Severity, strings.TrimSpace(record.Body))
			if err != nil {
				log.Error("failed to write deployment target log records to response writer", zap.Error(err))
				return
			}
			count++
		}
		if !written {
			SetFileDownloadHeaders(w, filename)
		} else if subscription.MaxLogExportRows.IsReached(count) {
			_, _ = w.Write([]byte(exportLimitNotice))
		}
	}
}
