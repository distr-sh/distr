package handlers

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func DashboardRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.With(middleware.RequireVendorOrPartner, middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Get("/artifacts-by-customer", respond.Handler(getArtifactsByCustomer))
	})
}

func getArtifactsByCustomer(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	var customers []types.CustomerOrganizationWithUsage
	var err error
	if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
		customers, err = db.GetCustomerOrganizationsByPartnerOrgID(ctx, *partnerOrgID)
	} else {
		customers, err = db.GetCustomerOrganizationsByOrganizationID(ctx, *auth.CurrentOrgID())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get customers: %w", err)
	} else if artifacts, err := db.GetArtifactsByOrgID(ctx, *auth.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	} else {
		licensing := auth.CurrentOrg().HasFeature(types.FeatureLicensing)
		artifactIDs := make([]uuid.UUID, len(artifacts))
		for i, artifact := range artifacts {
			artifactIDs[i] = artifact.ID
		}
		result := make([]api.ArtifactsByCustomer, 0)
		for _, customer := range customers {
			customerRes := api.ArtifactsByCustomer{Customer: mapping.CustomerOrganizationToAPI(customer.CustomerOrganization)}
			pulls, err := db.GetLatestArtifactPullsOfCustomerOrganization(ctx, *auth.CurrentOrgID(), customer.ID, artifactIDs)
			if err != nil {
				return nil, fmt.Errorf("failed to get latest artifact pulls: %w", err)
			}
			for _, artifact := range artifacts {
				i := slices.IndexFunc(pulls, func(p types.CustomerArtifactPull) bool { return p.ArtifactID == artifact.ID })
				if i < 0 {
					continue
				}
				versions, err := customerArtifactVersions(ctx, artifact.ID, customer.ID, licensing)
				if err != nil {
					return nil, fmt.Errorf("failed to get versions for artifact: %w", err)
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
		return respond.JSON(result), nil
	}
}
