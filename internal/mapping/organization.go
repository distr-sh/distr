package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
)

func OrganizationToAPI(o types.Organization, billableUserCount, customerOrgCount int64) api.OrganizationResponse {
	return api.OrganizationResponse{
		Organization:                     o,
		SubscriptionLimits:               subscription.GetSubscriptionLimits(o.SubscriptionType),
		CurrentBillableUserAccountCount:  billableUserCount,
		CurrentCustomerOrganizationCount: customerOrgCount,
	}
}
