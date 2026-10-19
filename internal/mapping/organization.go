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

func OrganizationToCreateResponse(o types.Organization, inviteURL string) api.CreateOrganizationResponse {
	return api.CreateOrganizationResponse{ID: o.ID, InviteURL: inviteURL}
}

func CreateOrganizationRequestToInternal(r api.CreateOrganizationRequest) types.Organization {
	return types.Organization{
		Name:                        r.Name,
		Slug:                        r.Slug,
		DockerPreConnectScript:      r.DockerPreConnectScript,
		DockerPostConnectScript:     r.DockerPostConnectScript,
		DockerConnectScriptIsSudo:   r.DockerConnectScriptIsSudo,
		KubernetesPreConnectScript:  r.KubernetesPreConnectScript,
		KubernetesPostConnectScript: r.KubernetesPostConnectScript,
	}
}
