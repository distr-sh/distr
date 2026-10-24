package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func CustomerOrganizationToLens(customerOrg types.CustomerOrganization) api.CustomerLens {
	return api.CustomerLens{
		ID:   customerOrg.ID,
		Name: customerOrg.Name,
	}
}
