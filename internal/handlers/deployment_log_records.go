package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/limit"
	"github.com/distr-sh/distr/internal/logstore"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

const latestRevisionsForActiveResources = 5

func getDeploymentLogsResourcesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		deployment := internalctx.GetDeployment(ctx)
		authInfo := auth.Authentication.Require(ctx)
		org := authInfo.CurrentOrg()

		revisionIDs, err := db.GetLatestDeploymentRevisionIDs(ctx, deployment.ID, latestRevisionsForActiveResources)
		if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get deployment revisions", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		logStore := logstore.FromContext(ctx)
		active, archived, err := logStore.GetDeploymentLogRecordResources(ctx, org.ID, logstore.DeploymentLogResourcesQuery{
			DeploymentID:      deployment.ID,
			LatestRevisionIDs: revisionIDs,
			Start:             subscription.GetLogQueryWindowStart(org.SubscriptionType),
		})
		if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get log records", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			RespondJSON(w, mapping.DeploymentLogRecordResourcesToAPI(active, archived))
		}
	}
}

func exportDeploymentLogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)

		deployment := internalctx.GetDeployment(ctx)

		resources := r.URL.Query()["resource"]
		if len(resources) == 0 {
			http.Error(w, "query parameter resource is required", http.StatusBadRequest)
			return
		}

		authInfo := auth.Authentication.Require(ctx)
		org := authInfo.CurrentOrg()

		queryRange, err := parseLogQueryRange(r, org.SubscriptionType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		filename := fmt.Sprintf("%s_%s.log", time.Now().Format("2006-01-02"), strings.Join(resources, "_"))

		var secrets []types.SecretWithUpdatedBy
		if dt, err := db.GetDeploymentTargetForDeploymentID(ctx, deployment.ID); err != nil {
			log.Error("failed to get deployment target", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if secrets, err = db.GetSecretsForDeploymentTarget(ctx, dt.DeploymentTarget); err != nil {
			log.Error("failed to get secrets", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		replacer := secretReplacer(secrets)
		export := newExportWriter(w, log, filename)

		if queryRange.IsEmpty() {
			export.finish()
			return
		}

		logStore := logstore.FromContext(ctx)
		records := logStore.QueryDeploymentLogRecords(ctx, org.ID, logstore.DeploymentLogQuery{
			DeploymentID: deployment.ID,
			Resources:    resources,
			Start:        queryRange.Start,
			End:          queryRange.End,
			Filter:       queryRange.Filter,
			Limit:        subscription.MaxLogExportRows,
			Direction:    types.OrderDirectionDesc,
		})
		for record, err := range records {
			if err != nil {
				export.fail(ctx, "failed to export log records", err)
				return
			}
			if err := export.writeLine("[%s] [%s] %s\n",
				record.Timestamp.Format(time.RFC3339),
				record.Severity,
				replacer.Replace(record.Body)); err != nil {
				return
			}
		}
		export.finish()
	}
}

func getDeploymentLogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		deployment := internalctx.GetDeployment(ctx)
		resources := r.URL.Query()["resource"]
		if len(resources) == 0 {
			http.Error(w, "query parameter resource is required", http.StatusBadRequest)
			return
		}
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
			RespondJSON(w, []api.DeploymentLogRecord{})
			return
		}

		var secrets []types.SecretWithUpdatedBy
		if dt, err := db.GetDeploymentTargetForDeploymentID(ctx, deployment.ID); err != nil {
			internalctx.GetLogger(ctx).Error("failed to get deployment target", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if secrets, err = db.GetSecretsForDeploymentTarget(ctx, dt.DeploymentTarget); err != nil {
			internalctx.GetLogger(ctx).Error("failed to get secrets", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		logStore := logstore.FromContext(ctx)
		if records, err := util.SeqCollect(logStore.QueryDeploymentLogRecords(ctx, org.ID, logstore.DeploymentLogQuery{
			DeploymentID: deployment.ID,
			Resources:    resources,
			Start:        queryRange.Start,
			End:          queryRange.End,
			Filter:       queryRange.Filter,
			Limit:        limit.Limit(limitParam),
			Direction:    direction,
		})); err != nil {
			if errors.Is(err, apierrors.ErrBadRequest) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			internalctx.GetLogger(ctx).Error("failed to get log records", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			replacer := secretReplacer(secrets)
			response := make([]api.DeploymentLogRecord, len(records))
			for i, record := range records {
				response[i] = api.DeploymentLogRecord{
					ID:                   record.ID,
					DeploymentID:         record.DeploymentID,
					DeploymentRevisionID: record.DeploymentRevisionID,
					Resource:             record.Resource,
					Timestamp:            record.Timestamp,
					Severity:             record.Severity,
					Body:                 replacer.Replace(record.Body),
				}
			}
			RespondJSON(w, response)
		}
	}
}

func secretReplacer(secrets []types.SecretWithUpdatedBy) *strings.Replacer {
	pairs := make([]string, 0, 2*len(secrets))
	for _, secret := range secrets {
		if secret.Value == "" {
			continue
		}
		pairs = append(pairs, secret.Value, "********")
	}
	if len(pairs) == 0 {
		return strings.NewReplacer()
	}
	return strings.NewReplacer(pairs...)
}
