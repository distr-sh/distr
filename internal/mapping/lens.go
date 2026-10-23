package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func OrganizationLensesToAPI(orgID uuid.UUID, customerOrgs []types.CustomerOrganization) api.OrganizationLenses {
	return api.OrganizationLenses{
		ID:                    orgID,
		CustomerOrganizations: List(customerOrgs, CustomerOrganizationToLens),
	}
}

func CustomerOrganizationToLens(customerOrg types.CustomerOrganization) api.CustomerLens {
	return api.CustomerLens{
		ID:   customerOrg.ID,
		Name: customerOrg.Name,
	}
}
