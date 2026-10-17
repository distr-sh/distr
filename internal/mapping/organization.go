package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
)

func OrganizationToAPI(
	o types.Organization,
	billableUserCount, customerOrgCount int64,
	defaultControllerImageRepository string,
	defaultControllerImageRepositorySource types.ControllerImageRepositorySource,
) api.OrganizationResponse {
	return api.OrganizationResponse{
		Organization:                           o,
		SubscriptionLimits:                     subscription.GetSubscriptionLimits(o.SubscriptionType),
		CurrentBillableUserAccountCount:        billableUserCount,
		CurrentCustomerOrganizationCount:       customerOrgCount,
		ControllerImageRepository:              o.ControllerImageRepository,
		DefaultControllerImageRepository:       defaultControllerImageRepository,
		DefaultControllerImageRepositorySource: defaultControllerImageRepositorySource,
		ControllerImageCustomDomain:            env.ControllerImageRepositoryUseCustomDomain(),
	}
}
