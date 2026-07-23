package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
)

type CheckoutResponse struct {
	SessionID string `json:"sessionId"`
	URL       string `json:"url"`
}

type CreateSubscriptionRequest struct {
	SubscriptionType        types.SubscriptionType   `json:"subscriptionType"`
	SubscriptionPeriod      types.SubscriptionPeriod `json:"subscriptionPeriod"`
	CustomerOrganizationQty int64                    `json:"subscriptionCustomerOrganizationQuantity"`
	UserAccountQty          int64                    `json:"subscriptionUserAccountQuantity"`
}

type UpdateSubscriptionRequest struct {
	// SubscriptionType optionally switches the subscription to a different plan.
	// Currently only the pro → business upgrade is supported.
	SubscriptionType        *types.SubscriptionType `json:"subscriptionType,omitempty"`
	CustomerOrganizationQty int64                   `json:"subscriptionCustomerOrganizationQuantity"`
	UserAccountQty          int64                   `json:"subscriptionUserAccountQuantity"`
}

type SubscriptionLimits struct {
	MaxCustomerOrganizations        int64 `json:"maxCustomerOrganizations"`
	MaxUsersPerCustomerOrganization int64 `json:"maxUsersPerCustomerOrganization"`
	MaxDeploymentsPerCustomerOrg    int64 `json:"maxDeploymentsPerCustomerOrganization"`
	// LogQueryWindowSeconds is how far back (in seconds) log read queries may reach.
	// The frontend uses it to constrain the log viewer's date pickers.
	LogQueryWindowSeconds int64 `json:"logQueryWindowSeconds"`
}

type SubscriptionInfo struct {
	SubscriptionType                       types.SubscriptionType   `json:"subscriptionType"`
	SubscriptionPeriod                     types.SubscriptionPeriod `json:"subscriptionPeriod"`
	SubscriptionEndsAt                     time.Time                `json:"subscriptionEndsAt"`
	SubscriptionCustomerOrganizationQty    int64                    `json:"subscriptionCustomerOrganizationQuantity"`
	SubscriptionUserAccountQty             int64                    `json:"subscriptionUserAccountQuantity"`
	CurrentUserAccountCount                int64                    `json:"currentUserAccountCount"`
	CurrentCustomerOrganizationCount       int64                    `json:"currentCustomerOrganizationCount"`
	CurrentMaxUsersPerCustomer             int64                    `json:"currentMaxUsersPerCustomer"`
	CurrentMaxDeploymentTargetsPerCustomer int64                    `json:"currentMaxDeploymentTargetsPerCustomer"`
	HasApplicationEntitlements             bool                     `json:"hasApplicationEntitlements"`
	HasArtifactEntitlements                bool                     `json:"hasArtifactEntitlements"`
	HasNonAdminRoles                       bool                     `json:"hasNonAdminRoles"`
	HasAlertConfigurations                 bool                     `json:"hasAlertConfigurations"`

	Limits map[types.SubscriptionType]SubscriptionLimits `json:"limits"`
}
