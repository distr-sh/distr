package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/handlerutil"
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
			Start:             time.Now().Add(-subscription.GetLogQueryWindow(org.SubscriptionType)),
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
		limit := int(subscription.GetLogExportRowsLimit(org.SubscriptionType))

		filename := fmt.Sprintf("%s_%s.log", time.Now().Format("2006-01-02"), strings.Join(resources, "_"))

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

		replacer := secretReplacer(secrets)

		logStore := logstore.FromContext(ctx)
		records := logStore.QueryDeploymentLogRecords(ctx, org.ID, logstore.DeploymentLogQuery{
			DeploymentID: deployment.ID,
			Resources:    resources,
			Start:        time.Now().Add(-subscription.GetLogQueryWindow(org.SubscriptionType)),
			Limit:        limit,
			Direction:    types.OrderDirectionDesc,
		})
		// The download headers are only set right before the first write, so an error
		// response can still be sent as long as nothing has been written yet.
		written := false
		for record, err := range records {
			if err != nil {
				log.Error("failed to export log records", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				if !written {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
				return
			}
			if !written {
				SetFileDownloadHeaders(w, filename)
				written = true
			}
			_, err := fmt.Fprintf(w, "[%s] [%s] %s\n",
				record.Timestamp.Format(time.RFC3339),
				record.Severity,
				replacer.Replace(record.Body))
			if err != nil {
				log.Error("failed to write log records to response writer", zap.Error(err))
				return
			}
		}
		if !written {
			SetFileDownloadHeaders(w, filename)
		}
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
			if err := handlerutil.ValidateFilterRegex(filter); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		order := types.OrderDirection(r.FormValue("order"))

		authInfo := auth.Authentication.Require(ctx)
		org := authInfo.CurrentOrg()
		// The effective direction must be resolved from the client-supplied "after"
		// before it is defaulted to the query window start below.
		direction := types.EffectiveOrderDirection(order, !after.IsZero())
		after, err = resolveLogQueryStart(org.SubscriptionType, after)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if before.IsZero() {
			before = time.Now()
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
			Start:        after,
			End:          before,
			Filter:       filter,
			Limit:        limit,
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
