package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

func getDeploymentTargetLogRecordsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		deploymentTarget := internalctx.GetDeploymentTarget(ctx)

		limit, err := QueryParam(r, "limit", strconv.Atoi, Max(100))
		if errors.Is(err, ErrParamNotDefined) {
			limit = 25
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		before, err := QueryParam(r, "before", ParseTimeFunc(time.RFC3339Nano))
		if err != nil && !errors.Is(err, ErrParamNotDefined) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		after, err := QueryParam(r, "after", ParseTimeFunc(time.RFC3339Nano))
		if err != nil && !errors.Is(err, ErrParamNotDefined) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		filter := r.FormValue("filter")
		if filter != "" {
			if _, err := regexp.Compile(filter); err != nil {
				http.Error(w, "invalid filter regex: "+err.Error(), http.StatusBadRequest)
				return
			}
		}

		records, err := db.GetDeploymentTargetLogRecords(ctx, deploymentTarget.ID, limit, before, after, filter)
		if err != nil {
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
		limit := int(subscription.GetLogExportRowsLimit(org.SubscriptionType))

		filename := fmt.Sprintf("%s_agent.log", time.Now().Format("2006-01-02"))

		SetFileDownloadHeaders(w, filename)

		records, err := db.GetDeploymentTargetLogRecordsSeq(ctx, deploymentTarget.ID, limit)
		if err != nil {
			log.Error("failed to export deployment target log records", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		for record, err := range records {
			if err != nil {
				log.Error("failed to export deployment target log records", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			_, err := fmt.Fprintf(w, "%s\t%s\t%s\n",
				record.Timestamp.Format(time.RFC3339), record.Severity, strings.TrimSpace(record.Body))
			if err != nil {
				log.Error("failed to write deployment target log records to response writer", zap.Error(err))
				return
			}
		}
	}
}
