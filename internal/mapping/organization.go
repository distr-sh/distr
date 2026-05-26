package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
)

func OrganizationToAPI(o types.Organization) api.OrganizationResponse {
	return api.OrganizationResponse{
		Organization:       o,
		SubscriptionLimits: subscription.GetSubscriptionLimits(o.SubscriptionType),
	}
}

// OrganizationWithRoleToAPI returns the OrganizationWithRole with the deprecated UserRole alias
// populated, ready to be serialized as JSON.
func OrganizationWithRoleToAPI(o types.OrganizationWithRole) types.OrganizationWithRole {
	o.PopulateDeprecatedAliases()
	return o
}
