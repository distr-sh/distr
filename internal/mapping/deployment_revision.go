package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

// organizationKindRank ranks the organization kinds by how restricted their
// visibility is: vendor (0) is the least restricted, customer (2) the most.
// A user is a customer when they have a customer organization, a partner when
// they have a partner organization and a vendor otherwise.
func organizationKindRank(customerOrgID, partnerOrgID *uuid.UUID) int {
	switch {
	case customerOrgID != nil:
		return 2
	case partnerOrgID != nil:
		return 1
	default:
		return 0
	}
}

// deploymentRevisionCreatorToAPI returns a mapper that converts the creator of a deployment revision to its
// API representation, or to nil when there is none. The creator's identity (id, name, email) is only included
// when the viewer is allowed to see it: vendors see everyone, partners see partners and customers and
// customers see only customers. A creator whose identity is withheld keeps their organization, unless they
// are deleted: then the viewer gets nil, as if there were no creator. A creator the viewer may see carries
// the deleted flag.
func deploymentRevisionCreatorToAPI(
	viewerCustomerOrgID *uuid.UUID,
	viewerPartnerOrgID *uuid.UUID,
) func(*types.DeploymentRevisionCreator) *api.DeploymentRevisionCreator {
	viewerRank := organizationKindRank(viewerCustomerOrgID, viewerPartnerOrgID)
	return func(c *types.DeploymentRevisionCreator) *api.DeploymentRevisionCreator {
		if c == nil {
			return nil
		}
		showIdentity := organizationKindRank(c.CustomerOrganizationID, c.PartnerOrganizationID) >= viewerRank
		if c.Deleted && !showIdentity {
			return nil
		}
		creator := &api.DeploymentRevisionCreator{
			CustomerOrganizationID: c.CustomerOrganizationID,
			PartnerOrganizationID:  c.PartnerOrganizationID,
			Deleted:                c.Deleted,
		}
		if showIdentity {
			creator.ID = &c.ID
			creator.ImageID = c.ImageID
			creator.Email = c.Email
			if c.Name != nil {
				creator.Name = *c.Name
			}
		}
		return creator
	}
}

// DeploymentRevisionToAPI returns a mapper that converts a DeploymentRevisionWithCreator to its API
// representation, showing the creator as deploymentRevisionCreatorToAPI does.
func DeploymentRevisionToAPI(
	viewerCustomerOrgID *uuid.UUID,
	viewerPartnerOrgID *uuid.UUID,
) func(types.DeploymentRevisionWithCreator) *api.DeploymentRevisionResponse {
	creatorToAPI := deploymentRevisionCreatorToAPI(viewerCustomerOrgID, viewerPartnerOrgID)
	return func(r types.DeploymentRevisionWithCreator) *api.DeploymentRevisionResponse {
		response := &api.DeploymentRevisionResponse{
			ID:                     r.ID,
			CreatedAt:              r.CreatedAt,
			ApplicationVersionID:   r.ApplicationVersionID,
			ApplicationVersionName: r.ApplicationVersionName,
			ReleaseName:            r.ReleaseName,
			DockerType:             r.DockerType,
			ValuesYaml:             r.ValuesYaml,
			EnvFileData:            r.EnvFileData,
			ForceRestart:           r.ForceRestart,
			IgnoreRevisionSkew:     r.IgnoreRevisionSkew,
			HelmOptions:            PtrOrNil(r.HelmOptions, HelmOptionsToAPI),
			Trigger:                r.Trigger,
			CreatedBy:              creatorToAPI(r.CreatedBy),
		}

		if r.LatestStatus != nil {
			response.LatestStatus = new(DeploymentRevisionStatusToAPI(*r.LatestStatus))
		}

		return response
	}
}
