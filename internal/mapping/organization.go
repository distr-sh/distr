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

func OrganizationMemberToAPI(member types.OrganizationMember) api.OrganizationMember {
	return api.OrganizationMember{
		ID:    member.ID,
		Email: member.Email,
		Name:  member.Name,
	}
}
