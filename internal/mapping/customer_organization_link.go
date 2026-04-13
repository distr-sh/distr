package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func CustomerOrganizationLinkToAPI(link types.CustomerOrganizationLink) api.CustomerOrganizationLink {
	return api.CustomerOrganizationLink{
		ID:                     link.ID,
		CreatedAt:              link.CreatedAt,
		CustomerOrganizationID: link.CustomerOrganizationID,
		Name:                   link.Name,
		Link:                   link.Link,
	}
}
