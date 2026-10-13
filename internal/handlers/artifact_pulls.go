package handlers

import (
	"encoding/csv"
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
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
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
		middleware.RequireVendorOrPartner,
	)
	r.Get("/", respond.Handler(getArtifactPullsHandler)).
		With(option.Description("List artifact version pulls")).
		With(option.Request(struct {
			Before                 *time.Time `query:"before"`
			After                  *time.Time `query:"after"`
			Count                  *int       `query:"count"`
			CustomerOrganizationID *string    `query:"customerOrganizationId"`
			UserAccountID          *string    `query:"userAccountId"`
			Anonymous              *bool      `query:"anonymous"`
			RemoteAddress          *string    `query:"remoteAddress"`
			ArtifactID             *string    `query:"artifactId"`
			ArtifactVersionID      *string    `query:"artifactVersionId"`
			DeploymentTargetID     *string    `query:"deploymentTargetId"`
		}{})).
		With(option.Response(http.StatusOK, []api.ArtifactVersionPullResponse{}))
	r.Get("/filter-options", respond.Handler(getArtifactPullFilterOptionsHandler)).
		With(option.Description("Get available filter options for artifact pulls")).
		With(option.Response(http.StatusOK, api.ArtifactVersionPullFilterOptions{}))
	r.Get("/filter-options/versions", respond.Handler(getArtifactPullVersionOptionsHandler)).
		With(option.Description("Get available version filter options for a specific artifact")).
		With(option.Request(struct {
			ArtifactID string `query:"artifactId"`
		}{})).
		With(option.Response(http.StatusOK, []api.ArtifactPullFilterOption{}))
	r.Get("/export", exportArtifactPullsHandler()).
		With(option.Description("Export artifact version pulls as CSV")).
		With(option.Response(http.StatusOK, []byte{}))
}

// parseArtifactPullFilters is artifactPullFilter for handlers that write to w themselves.
func parseArtifactPullFilters(w http.ResponseWriter, r *http.Request) (types.ArtifactVersionPullFilter, error) {
	filter, err := artifactPullFilter(r)
	if err != nil {
		resp, respErr := artifactPullFilterError(err)
		respond.Write(w, r, resp, respErr)
	}
	return filter, err
}

func artifactPullFilter(r *http.Request) (types.ArtifactVersionPullFilter, error) {
	ctx := r.Context()
	authInfo := auth.Authentication.Require(ctx)
	filter := types.ArtifactVersionPullFilter{
		OrgID:  *authInfo.CurrentOrgID(),
		Before: time.Now(),
		Count:  50,
	}

	if before, err := QueryParam(r, "before", ParseTimeFunc(time.RFC3339Nano)); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, apierrors.NewBadRequest("before must be a valid date")
	} else {
		filter.Before = before
	}

	if after, err := QueryParam(r, "after", ParseTimeFunc(time.RFC3339Nano)); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, apierrors.NewBadRequest("after must be a valid date")
	} else {
		filter.After = after
	}

	if count, err := QueryParam(r, "count", strconv.Atoi); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, apierrors.NewBadRequest("count must be a number")
	} else if count < 1 || count > 1000 {
		return filter, apierrors.NewBadRequest("count must be between 1 and 1000")
	} else {
		filter.Count = count
	}

	if id, err := QueryParam(r, "customerOrganizationId", uuid.Parse); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, apierrors.NewBadRequest("customerOrganizationId must be a valid UUID")
	} else {
		filter.CustomerOrganizationID = &id
	}

	if id, err := QueryParam(r, "userAccountId", uuid.Parse); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, apierrors.NewBadRequest("userAccountId must be a valid UUID")
	} else {
		filter.UserAccountID = &id
	}

	if anonymous, err := QueryParam(r, "anonymous", strconv.ParseBool); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, apierrors.NewBadRequest("anonymous must be true or false")
	} else {
		filter.Anonymous = anonymous
	}

	if addr := strings.TrimSpace(r.FormValue("remoteAddress")); addr != "" {
		filter.RemoteAddress = &addr
	}

	if id, err := QueryParam(r, "artifactId", uuid.Parse); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, apierrors.NewBadRequest("artifactId must be a valid UUID")
	} else {
		filter.ArtifactID = &id
	}

	if id, err := QueryParam(r, "artifactVersionId", uuid.Parse); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, apierrors.NewBadRequest("artifactVersionId must be a valid UUID")
	} else {
		filter.ArtifactVersionID = &id
	}

	if id, err := QueryParam(r, "deploymentTargetId", uuid.Parse); errors.Is(err, ErrParamNotDefined) {
		// use default
	} else if err != nil {
		return filter, apierrors.NewBadRequest("deploymentTargetId must be a valid UUID")
	} else {
		filter.DeploymentTargetID = &id
	}

	if partnerOrgID := authInfo.CurrentPartnerOrgID(); partnerOrgID != nil {
		filter.PartnerOrganizationID = partnerOrgID
		if filter.CustomerOrganizationID != nil {
			co, err := db.GetCustomerOrganizationByID(ctx, *filter.CustomerOrganizationID)
			if err != nil {
				if errors.Is(err, apierrors.ErrNotFound) {
					return filter, apierrors.NewForbidden("customer is not assigned to your partner organization")
				}
				return filter, err
			}
			if !util.PtrEq(co.PartnerOrganizationID, partnerOrgID) {
				return filter, apierrors.NewForbidden("customer is not assigned to your partner organization")
			}
		}
	}

	return filter, nil
}

func artifactPullFilterError(err error) (respond.Responder, error) {
	switch {
	case errors.Is(err, apierrors.ErrBadRequest):
		return respond.BadRequest(err.Error()), nil
	case errors.Is(err, apierrors.ErrForbidden):
		return respond.Forbidden(err.Error()), nil
	default:
		return nil, fmt.Errorf("could not parse artifact pull filters: %w", err)
	}
}

func getArtifactPullsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()

	if filter, err := artifactPullFilter(r); err != nil {
		return artifactPullFilterError(err)
	} else if pulls, err := db.GetArtifactVersionPulls(ctx, filter); err != nil {
		return nil, fmt.Errorf("could not get pulls: %w", err)
	} else {
		return respond.JSON(mapping.List(pulls, mapping.ArtifactVersionPullToAPI)), nil
	}
}

func getArtifactPullFilterOptionsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authInfo := auth.Authentication.Require(ctx)

	if opts, err := db.GetArtifactVersionPullFilterOptions(
		ctx, *authInfo.CurrentOrgID(), authInfo.CurrentPartnerOrgID(),
	); err != nil {
		return nil, fmt.Errorf("could not get pull filter options: %w", err)
	} else {
		return respond.JSON(mapping.ArtifactVersionPullFilterOptionsToAPI(opts)), nil
	}
}

func getArtifactPullVersionOptionsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authInfo := auth.Authentication.Require(ctx)

	if artifactID, err := QueryParam(r, "artifactId", uuid.Parse); err != nil {
		return respond.BadRequest("artifactId is required and must be a valid UUID"), nil
	} else if versions, err := db.GetArtifactVersionPullVersionOptions(
		ctx,
		*authInfo.CurrentOrgID(),
		artifactID,
		authInfo.CurrentPartnerOrgID(),
	); err != nil {
		return nil, fmt.Errorf("could not get version filter options: %w", err)
	} else {
		return respond.JSON(mapping.List(versions, mapping.FilterOptionToAPI)), nil
	}
}

func exportArtifactPullsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)

		filter, err := parseArtifactPullFilters(w, r)
		if err != nil {
			return
		}

		filter.Count = int(subscription.MaxArtifactPullExportRows)

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
		header := []string{
			"Date", "Customer", "User", "Email", "Deployment Target", "Anonymous", "Address",
			"Artifact", "Version",
		}
		if err := csvWriter.Write(header); err != nil {
			log.Warn("could not write CSV header", zap.Error(err))
			return
		}

		for _, pull := range pulls {
			apiPull := mapping.ArtifactVersionPullToAPI(pull)
			if err := csvWriter.Write([]string{
				apiPull.CreatedAt.Format(time.RFC3339),
				util.PtrDerefOrDefault(apiPull.CustomerOrganizationName),
				util.PtrDerefOrDefault(apiPull.UserAccountName),
				util.PtrDerefOrDefault(apiPull.UserAccountEmail),
				util.PtrDerefOrDefault(apiPull.DeploymentTargetName),
				strconv.FormatBool(apiPull.Anonymous),
				util.PtrDerefOrDefault(apiPull.RemoteAddress),
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
