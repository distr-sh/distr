package handlers

import (
	"net/http"
	"slices"

	"github.com/distr-sh/distr/api"
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
		licensing := auth.CurrentOrg().HasFeature(types.FeatureLicensing)
		artifactIDs := make([]uuid.UUID, len(artifacts))
		for i, artifact := range artifacts {
			artifactIDs[i] = artifact.ID
		}
		result := make([]api.ArtifactsByCustomer, 0)
		for _, customer := range customers {
			customerRes := api.ArtifactsByCustomer{Customer: mapping.CustomerOrganizationToAPI(customer.CustomerOrganization)}
			pulls, err := db.GetLatestArtifactPullsOfCustomerOrganization(
				ctx, *auth.CurrentOrgID(), customer.ID, artifactIDs,
			)
			if err != nil {
				respondInternalError(w, r, err, "failed to get latest artifact pulls")
				return
			}
			for _, artifact := range artifacts {
				i := slices.IndexFunc(pulls, func(p types.CustomerArtifactPull) bool { return p.ArtifactID == artifact.ID })
				if i < 0 {
					continue
				}
				versions, err := customerArtifactVersions(ctx, artifact.ID, customer.ID, licensing)
				if err != nil {
					respondInternalError(w, r, err, "failed to get versions for artifact")
					return
				}
				latest, _ := types.LatestArtifactTag(versions)
				customerRes.Artifacts = append(customerRes.Artifacts, api.DashboardArtifact{
					Artifact: mapping.ArtifactToAPI(types.ArtifactWithTaggedVersion{
						ArtifactWithDownloads: artifact,
						Versions:              versions,
					}),
					LatestPulledVersion: pulls[i].VersionName,
					IsLatest:            types.ArtifactPullIsLatest(versions, latest, pulls[i].VersionName),
				})
			}

			result = append(result, customerRes)
		}

		RespondJSON(w, result)
	}
}
