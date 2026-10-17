package handlers

import (
	"errors"
	"fmt"
	"iter"
	"net/http"
	"strings"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/limit"
	"github.com/distr-sh/distr/internal/logstore"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
)

const latestRevisionsForActiveResources = 5

func getDeploymentLogsResourcesHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	deployment := internalctx.GetDeployment(ctx)
	authInfo := auth.Authentication.Require(ctx)
	org := authInfo.CurrentOrg()

	if revisionIDs, err := db.GetLatestDeploymentRevisionIDs(
		ctx, deployment.ID, latestRevisionsForActiveResources,
	); err != nil {
		return nil, fmt.Errorf("failed to get deployment revisions: %w", err)
	} else if active, archived, err := logstore.FromContext(ctx).GetDeploymentLogRecordResources(
		ctx, org.ID, logstore.DeploymentLogResourcesQuery{
			DeploymentID:      deployment.ID,
			LatestRevisionIDs: revisionIDs,
			Start:             subscription.GetLogQueryWindowStart(org.SubscriptionType),
		},
	); err != nil {
		return nil, fmt.Errorf("failed to get log records: %w", err)
	} else {
		return respond.JSON(mapping.DeploymentLogRecordResourcesToAPI(active, archived)), nil
	}
}

func exportDeploymentLogsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	deployment := internalctx.GetDeployment(ctx)
	resources := r.URL.Query()["resource"]
	org := auth.Authentication.Require(ctx).CurrentOrg()

	if len(resources) == 0 {
		return respond.BadRequest("query parameter resource is required"), nil
	} else if queryRange, err := parseLogQueryRange(r, org.SubscriptionType); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if dt, err := db.GetDeploymentTargetForDeploymentID(ctx, deployment.ID); err != nil {
		return nil, fmt.Errorf("failed to get deployment target: %w", err)
	} else if secrets, err := db.GetSecretsForDeploymentTarget(ctx, dt.DeploymentTarget); err != nil {
		return nil, fmt.Errorf("failed to get secrets: %w", err)
	} else {
		var records iter.Seq2[logstore.DeploymentLogRecord, error] = noRecords
		if !queryRange.IsEmpty() {
			records = logstore.FromContext(ctx).QueryDeploymentLogRecords(ctx, org.ID, logstore.DeploymentLogQuery{
				DeploymentID: deployment.ID,
				Resources:    resources,
				Start:        queryRange.Start,
				End:          queryRange.End,
				Filter:       queryRange.Filter,
				Limit:        subscription.MaxLogExportRows,
				Direction:    types.OrderDirectionDesc,
			})
		}

		filename := fmt.Sprintf("%s_%s.log", time.Now().Format("2006-01-02"), strings.Join(resources, "_"))
		replacer := secretReplacer(secrets)
		return logExport(filename, records, func(record logstore.DeploymentLogRecord) string {
			return fmt.Sprintf("[%s] [%s] %s\n",
				record.Timestamp.Format(time.RFC3339), record.Severity, replacer.Replace(record.Body))
		}), nil
	}
}

func getDeploymentLogsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	deployment := internalctx.GetDeployment(ctx)
	resources := r.URL.Query()["resource"]
	order := types.OrderDirection(r.FormValue("order"))
	authInfo := auth.Authentication.Require(ctx)
	org := authInfo.CurrentOrg()

	if len(resources) == 0 {
		return respond.BadRequest("query parameter resource is required"), nil
	} else if limitParam, err := parseTimeseriesLimit(r); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if queryRange, err := parseLogQueryRange(r, org.SubscriptionType); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if queryRange.IsEmpty() {
		return respond.JSON([]api.DeploymentLogRecord{}), nil
	} else if dt, err := db.GetDeploymentTargetForDeploymentID(ctx, deployment.ID); err != nil {
		return nil, fmt.Errorf("failed to get deployment target: %w", err)
	} else if secrets, err := db.GetSecretsForDeploymentTarget(ctx, dt.DeploymentTarget); err != nil {
		return nil, fmt.Errorf("failed to get secrets: %w", err)
	} else if records, err := util.SeqCollect(logstore.FromContext(ctx).QueryDeploymentLogRecords(
		ctx, org.ID, logstore.DeploymentLogQuery{
			DeploymentID: deployment.ID,
			Resources:    resources,
			Start:        queryRange.Start,
			End:          queryRange.End,
			Filter:       queryRange.Filter,
			Limit:        limit.Limit(limitParam),
			Direction:    types.EffectiveOrderDirection(order, queryRange.StartExplicit),
		},
	)); err != nil {
		if errors.Is(err, distrerr.ErrBadRequest) {
			return respond.BadRequest(err.Error()), nil
		} else {
			return nil, fmt.Errorf("failed to get log records: %w", err)
		}
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
		return respond.JSON(response), nil
	}
}

func secretReplacer(secrets []types.SecretWithUpdatedBy) *strings.Replacer {
	pairs := make([]string, 0, 2*len(secrets))
	for _, secret := range secrets {
		if secret.Value == "" {
			continue
		}
		pairs = append(pairs, string(secret.Value), "********")
	}
	if len(pairs) == 0 {
		return strings.NewReplacer()
	}
	return strings.NewReplacer(pairs...)
}
