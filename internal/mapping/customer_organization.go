package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func CustomerOrganizationToAPI(customerOrganization types.CustomerOrganization) api.CustomerOrganization {
	return api.CustomerOrganization{
		ID:        customerOrganization.ID,
		CreatedAt: customerOrganization.CreatedAt,
		Name:      customerOrganization.Name,
		ImageID:   customerOrganization.ImageID,
		ImageURL:  CreateImageURL(customerOrganization.ImageID),
		Features:  customerOrganization.Features,
	}
}

func CustomerOrganizationWithUsageToAPI(
	customerOrganizationWithUsage types.CustomerOrganizationWithUsage,
) api.CustomerOrganizationWithUsage {
	return api.CustomerOrganizationWithUsage{
		CustomerOrganization:  CustomerOrganizationToAPI(customerOrganizationWithUsage.CustomerOrganization),
		UserCount:             customerOrganizationWithUsage.UserCount,
		DeploymentTargetCount: customerOrganizationWithUsage.DeploymentTargetCount,
	}
}
