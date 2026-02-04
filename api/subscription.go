package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
)

type CheckoutResponse struct {
	SessionID string `json:"sessionId"`
	URL       string `json:"url"`
}

type SubscriptionLimits struct {
	MaxCustomerOrganizations        int64 `json:"maxCustomerOrganizations"`
	MaxUsersPerCustomerOrganization int64 `json:"maxUsersPerCustomerOrganization"`
	MaxDeploymentsPerCustomerOrg    int64 `json:"maxDeploymentsPerCustomerOrganization"`
}

type SubscriptionInfo struct {
	SubscriptionType                              types.SubscriptionType   `json:"subscriptionType"`
	SubscriptionPeriod                            types.SubscriptionPeriod `json:"subscriptionPeriod"`
	SubscriptionEndsAt                            time.Time                `json:"subscriptionEndsAt"`
	SubscriptionCustomerOrganizationQty           int64                    `json:"subscriptionCustomerOrganizationQuantity"` //nolint:lll
	SubscriptionUserAccountQty                    int64                    `json:"subscriptionUserAccountQuantity"`
	CurrentUserAccountCount                       int64                    `json:"currentUserAccountCount"`
	CurrentCustomerOrganizationCount              int64                    `json:"currentCustomerOrganizationCount"`
	CurrentMaxUsersPerCustomer                    int64                    `json:"currentMaxUsersPerCustomer"`
	CurrentMaxDeploymentTargetsPerCustomer        int64                    `json:"currentMaxDeploymentTargetsPerCustomer"`
	HasApplicationLicenses                        bool                     `json:"hasApplicationLicenses"`
	HasArtifactLicenses                           bool                     `json:"hasArtifactLicenses"`
	HasNonAdminRoles                              bool                     `json:"hasNonAdminRoles"`
	HasDeploymentStatusNotificationConfigurations bool                     `json:"hasDeploymentStatusNotificationConfigurations"` //nolint:lll

	Limits map[types.SubscriptionType]SubscriptionLimits `json:"limits"`
}
