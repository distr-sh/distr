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
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

func getDeploymentLogsResourcesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		deployment := internalctx.GetDeployment(ctx)
		if active, archived, err := db.GetDeploymentLogRecordResources(ctx, deployment.ID); err != nil {
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

		SetFileDownloadHeaders(w, filename)

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

		err := db.GetDeploymentLogRecordsForExport(
			ctx, deployment.ID, resources, limit,
			func(record types.DeploymentLogRecord) error {
				_, err := fmt.Fprintf(w, "[%s] [%s] %s\n",
					record.Timestamp.Format(time.RFC3339),
					record.Severity,
					replacer.Replace(record.Body))
				return err
			},
		)
		if err != nil {
			log.Error("failed to export log records", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			// Note: If headers were already sent, we can't send error response
			return
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
		order := types.ParseOrderDirection(r.FormValue("order"))

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

		if records, err := db.GetDeploymentLogRecords(
			ctx, deployment.ID, resources, limit, before, after, filter, order,
		); err != nil {
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
