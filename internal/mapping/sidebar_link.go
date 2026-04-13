package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func SidebarLinkToAPI(link types.SidebarLink) api.SidebarLink {
	return api.SidebarLink{
		ID:                     link.ID,
		CreatedAt:              link.CreatedAt,
		OrganizationID:         link.OrganizationID,
		CustomerOrganizationID: link.CustomerOrganizationID,
		Name:                   link.Name,
		Link:                   link.Link,
	}
}
