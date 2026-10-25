package handlers

import (
	"context"
	"fmt"
	"net/http"

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
		result := make([]api.ArtifactsByCustomer, len(customers))
		for i, customer := range customers {
			if pulledArtifacts, err := pulledArtifactsOfCustomer(
				ctx, artifacts, artifactIDs, customer.ID, licensing,
			); err != nil {
				return nil, err
			} else {
				result[i] = api.ArtifactsByCustomer{
					Customer:  mapping.CustomerOrganizationToAPI(customer.CustomerOrganization),
					Artifacts: pulledArtifacts,
				}
			}
		}
		return respond.JSON(result), nil
	}
}

// pulledArtifactsOfCustomer returns the artifacts the customer organization pulled, each with the version it
// pulled last and the versions it may pull.
func pulledArtifactsOfCustomer(
	ctx context.Context,
	artifacts []types.ArtifactWithDownloads,
	artifactIDs []uuid.UUID,
	customerOrgID uuid.UUID,
	licensing bool,
) ([]api.DashboardArtifact, error) {
	orgID := *auth.Authentication.Require(ctx).CurrentOrgID()
	pulls, err := db.GetLatestArtifactPullsOfCustomerOrganization(ctx, orgID, customerOrgID, artifactIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest artifact pulls: %w", err)
	} else if len(pulls) == 0 {
		return nil, nil
	}
	pullsByArtifact := make(map[uuid.UUID]types.ArtifactVersionPull, len(pulls))
	pulledArtifactIDs := make([]uuid.UUID, len(pulls))
	for i, pull := range pulls {
		pullsByArtifact[pull.Artifact.ID] = pull
		pulledArtifactIDs[i] = pull.Artifact.ID
	}
	releaseVersions, err := db.ListArtifactReleaseVersions(ctx, pulledArtifactIDs, customerOrgID, licensing)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact versions: %w", err)
	}

	versions := types.TaggedArtifactVersionsByArtifact(releaseVersions)
	var result []api.DashboardArtifact
	for _, artifact := range artifacts {
		pull, pulled := pullsByArtifact[artifact.ID]
		if !pulled {
			continue
		}
		latestPulledVersion := pull.ArtifactVersion.Name
		result = append(result, api.DashboardArtifact{
			Artifact: mapping.ArtifactToAPI(types.ArtifactWithTaggedVersion{
				ArtifactWithDownloads: artifact,
				Versions:              versions[artifact.ID],
			}),
			LatestPulledVersion: latestPulledVersion,
		})
	}
	return result, nil
}
