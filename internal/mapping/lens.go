package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func CustomerOrganizationToLens(customerOrg types.CustomerOrganization) api.CustomerLens {
	return api.CustomerLens{
		CustomerOrganizationID:   customerOrg.ID,
		CustomerOrganizationName: customerOrg.Name,
	}
}
