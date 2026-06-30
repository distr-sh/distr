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

// DeploymentRevisionToAPI returns a mapper that converts a
// DeploymentRevisionWithCreator to its API representation. The creator's
// identity (id, name, email) is only included when the viewer is allowed to see
// it: vendors see everyone, partners see partners and customers and customers
// see only customers. The creator's organization and deleted flag are always
// included when a creator is present.
func DeploymentRevisionToAPI(
	viewerCustomerOrgID *uuid.UUID,
	viewerPartnerOrgID *uuid.UUID,
) func(types.DeploymentRevisionWithCreator) *api.DeploymentRevisionResponse {
	viewerRank := organizationKindRank(viewerCustomerOrgID, viewerPartnerOrgID)
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
			HelmOptions:            r.HelmOptions,
		}

		if r.CreatedByID != nil {
			creatorRank := organizationKindRank(r.CreatedByCustomerOrganizationID, r.CreatedByPartnerOrganizationID)
			showIdentity := creatorRank >= viewerRank
			// A deleted creator is hidden entirely (CreatedBy stays nil) when the viewer is not
			// allowed to see its identity. Otherwise the creator is always represented, with the
			// identity withheld when the viewer is not allowed to see it.
			if !r.CreatedByDeleted || showIdentity {
				creator := &api.DeploymentRevisionCreator{
					CustomerOrganizationID: r.CreatedByCustomerOrganizationID,
					PartnerOrganizationID:  r.CreatedByPartnerOrganizationID,
					Deleted:                r.CreatedByDeleted,
				}
				if showIdentity {
					creator.ID = r.CreatedByID
					creator.ImageID = r.CreatedByImageID
					if r.CreatedByEmail != nil {
						creator.Email = *r.CreatedByEmail
					}
					if r.CreatedByName != nil {
						creator.Name = *r.CreatedByName
					}
				}
				response.CreatedBy = creator
			}
		}

		return response
	}
}
