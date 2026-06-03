package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func PartnerOrganizationToAPI(partnerOrg types.PartnerOrganization) api.PartnerOrganization {
	return api.PartnerOrganization{
		ID:        partnerOrg.ID,
		CreatedAt: partnerOrg.CreatedAt,
		Name:      partnerOrg.Name,
	}
}

func PartnerOrganizationWithUsageToAPI(partnerOrg types.PartnerOrganizationWithUsage) api.PartnerOrganizationWithUsage {
	return api.PartnerOrganizationWithUsage{
		PartnerOrganization:       PartnerOrganizationToAPI(partnerOrg.PartnerOrganization),
		UserCount:                 partnerOrg.UserCount,
		CustomerOrganizationCount: partnerOrg.CustomerOrganizationCount,
	}
}
