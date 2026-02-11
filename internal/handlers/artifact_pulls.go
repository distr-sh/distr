package handlers

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func ArtifactPullsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Artifacts"))
	r.Use(
		middleware.RequireOrgAndRole,
		middleware.RequireVendor,
	)
	r.Get("/", getArtifactPullsHandler()).
		With(option.Description("List artifact version pulls")).
		With(option.Request(struct {
			Before                 *time.Time `query:"before"`
			After                  *time.Time `query:"after"`
			Count                  *int       `query:"count"`
			CustomerOrganizationID *string    `query:"customerOrganizationId"`
			UserAccountID          *string    `query:"userAccountId"`
			RemoteAddress          *string    `query:"remoteAddress"`
			ArtifactID             *string    `query:"artifactId"`
			ArtifactVersionID      *string    `query:"artifactVersionId"`
		}{})).
		With(option.Response(http.StatusOK, []api.ArtifactVersionPullResponse{}))
	r.Get("/filter-options", getArtifactPullFilterOptionsHandler()).
		With(option.Description("Get available filter options for artifact pulls")).
		With(option.Response(http.StatusOK, api.ArtifactVersionPullFilterOptions{}))
	r.Get("/filter-options/versions", getArtifactPullVersionOptionsHandler()).
		With(option.Description("Get available version filter options for a specific artifact")).
		With(option.Request(struct {
			ArtifactID string `query:"artifactId"`
		}{})).
		With(option.Response(http.StatusOK, []api.ArtifactPullFilterOption{}))
	r.Get("/export", exportArtifactPullsHandler()).
		With(option.Description("Export artifact version pulls as CSV")).
		With(option.Response(http.StatusOK, []byte{}))
}

func parseArtifactPullFilters(r *http.Request, orgID uuid.UUID) (types.ArtifactVersionPullFilter, error) {
	filter := types.ArtifactVersionPullFilter{
		OrgID:  orgID,
		Before: time.Now(),
		Count:  50,
	}

	if before, err := QueryParam(r, "before", ParseTimeFunc(time.RFC3339Nano)); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, fmt.Errorf("before must be a valid date")
	} else {
		filter.Before = before
	}

	if after, err := QueryParam(r, "after", ParseTimeFunc(time.RFC3339Nano)); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, fmt.Errorf("after must be a valid date")
	} else {
		filter.After = after
	}

	if count, err := QueryParam(r, "count", strconv.Atoi); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, fmt.Errorf("count must be a number")
	} else if count < 1 || count > 1000 {
		return filter, fmt.Errorf("count must be between 1 and 1000")
	} else {
		filter.Count = count
	}

	if id, err := QueryParam(r, "customerOrganizationId", uuid.Parse); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, fmt.Errorf("customerOrganizationId must be a valid UUID")
	} else {
		filter.CustomerOrganizationID = &id
	}

	if id, err := QueryParam(r, "userAccountId", uuid.Parse); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, fmt.Errorf("userAccountId must be a valid UUID")
	} else {
		filter.UserAccountID = &id
	}

	if addr := r.FormValue("remoteAddress"); addr != "" {
		filter.RemoteAddress = &addr
	}

	if id, err := QueryParam(r, "artifactId", uuid.Parse); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, fmt.Errorf("artifactId must be a valid UUID")
	} else {
		filter.ArtifactID = &id
	}

	if id, err := QueryParam(r, "artifactVersionId", uuid.Parse); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, fmt.Errorf("artifactVersionId must be a valid UUID")
	} else {
		filter.ArtifactVersionID = &id
	}

	return filter, nil
}

func getArtifactPullsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		authInfo := auth.Authentication.Require(ctx)

		filter, err := parseArtifactPullFilters(r, *authInfo.CurrentOrgID())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		pulls, err := db.GetArtifactVersionPulls(ctx, filter)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Warn("could not get pulls", zap.Error(err))
			return
		}

		RespondJSON(w, mapping.List(pulls, mapping.ArtifactVersionPullToAPI))
	}
}

func getArtifactPullFilterOptionsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		authInfo := auth.Authentication.Require(ctx)

		opts, err := db.GetArtifactVersionPullFilterOptions(ctx, *authInfo.CurrentOrgID())
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Warn("could not get pull filter options", zap.Error(err))
			return
		}

		RespondJSON(w, mapping.ArtifactVersionPullFilterOptionsToAPI(opts))
	}
}

func getArtifactPullVersionOptionsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		authInfo := auth.Authentication.Require(ctx)

		artifactID, err := QueryParam(r, "artifactId", uuid.Parse)
		if err != nil {
			http.Error(w, "artifactId is required and must be a valid UUID", http.StatusBadRequest)
			return
		}

		versions, err := db.GetArtifactVersionPullVersionOptions(ctx, *authInfo.CurrentOrgID(), artifactID)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Warn("could not get version filter options", zap.Error(err))
			return
		}

		RespondJSON(w, mapping.List(versions, mapping.FilterOptionToAPI))
	}
}

func exportArtifactPullsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		authInfo := auth.Authentication.Require(ctx)
		org := authInfo.CurrentOrg()

		filter, err := parseArtifactPullFilters(r, *authInfo.CurrentOrgID())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		filter.Count = int(subscription.GetLogExportRowsLimit(org.SubscriptionType))

		pulls, err := db.GetArtifactVersionPulls(ctx, filter)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Warn("could not get pulls for export", zap.Error(err))
			return
		}

		filename := fmt.Sprintf("%s_artifact_pulls.csv", time.Now().Format("2006-01-02"))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

		csvWriter := csv.NewWriter(w)
		header := []string{"Date", "Customer", "User", "Email", "Address", "Artifact", "Version"}
		if err := csvWriter.Write(header); err != nil {
			log.Warn("could not write CSV header", zap.Error(err))
			return
		}

		for _, pull := range pulls {
			apiPull := mapping.ArtifactVersionPullToAPI(pull)
			if err := csvWriter.Write([]string{
				apiPull.CreatedAt.Format(time.RFC3339),
				derefOrDefault(apiPull.CustomerOrganizationName),
				derefOrDefault(apiPull.UserAccountName),
				derefOrDefault(apiPull.UserAccountEmail),
				derefOrDefault(apiPull.RemoteAddress),
				apiPull.Artifact.Name,
				apiPull.ArtifactVersion.Name,
			}); err != nil {
				log.Warn("could not write CSV row", zap.Error(err))
				return
			}
		}

		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			log.Warn("CSV flush error", zap.Error(err))
		}
	}
}

func derefOrDefault(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
