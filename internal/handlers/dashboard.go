package handlers

import (
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func DashboardRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.With(middleware.RequireVendorOrPartner, middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Get("/artifacts-by-customer", getArtifactsByCustomer)
	})
}

func getArtifactsByCustomer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	var customers []types.CustomerOrganizationWithUsage
	var err error
	if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
		customers, err = db.GetCustomerOrganizationsByPartnerOrgID(ctx, *partnerOrgID)
	} else {
		customers, err = db.GetCustomerOrganizationsByOrganizationID(ctx, *auth.CurrentOrgID())
	}
	if err != nil {
		log.Error("failed to get customers", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	} else if artifacts, err := db.GetArtifactsByOrgID(ctx, *auth.CurrentOrgID()); err != nil {
		log.Error("failed to get artifacts", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	} else {
		result := make([]api.ArtifactsByCustomer, 0)
		for _, customer := range customers {
			customerRes := api.ArtifactsByCustomer{Customer: mapping.CustomerOrganizationToAPI(customer.CustomerOrganization)}
			for _, artifact := range artifacts {
				if latestPulled, err := db.GetLatestPullOfArtifactByCustomerOrganization(
					ctx,
					artifact.ID,
					customer.ID,
				); err != nil {
					if errors.Is(err, apierrors.ErrNotFound) {
						continue
					} else {
						log.Error("failed to get latest artifact pull by user", zap.Error(err),
							zap.Any("artifactId", artifact.ID), zap.Any("userId", customer.ID))
						sentry.GetHubFromContext(ctx).CaptureException(err)
					}
				} else {
					var licenseOwnerID *uuid.UUID
					if auth.CurrentOrg().HasFeature(types.FeatureLicensing) {
						licenseOwnerID = &customer.ID
					}

					if versions, err := db.GetVersionsForArtifact(ctx, artifact.ID, licenseOwnerID); err != nil {
						log.Error("failed to get versions for artifact", zap.Error(err))
						sentry.GetHubFromContext(ctx).CaptureException(err)
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
					} else {
						customerRes.Artifacts = append(customerRes.Artifacts, api.DashboardArtifact{
							Artifact: mapping.ArtifactToAPI(types.ArtifactWithTaggedVersion{
								ArtifactWithDownloads: artifact,
								Versions:              versions,
							}),
							LatestPulledVersion: latestPulled,
						})
					}
				}
			}

			result = append(result, customerRes)
		}

		RespondJSON(w, result)
	}
}
