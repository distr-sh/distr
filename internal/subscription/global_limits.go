package subscription

import (
	"fmt"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/license"
	"github.com/distr-sh/distr/internal/limit"
	"github.com/distr-sh/distr/internal/types"
)

const (
	MaxCustomersPerOrganizationCommunity             = limit.Unlimited
	MaxCustomersPerOrganizationStarter   limit.Limit = 3
	MaxCustomersPerOrganizationPro       limit.Limit = 100
	MaxCustomersPerOrganizationTrial                 = limit.Unlimited

	MaxUsersPerCustomerOrganizationCommunity limit.Limit = 1
	MaxUsersPerCustomerOrganizationStarter   limit.Limit = 1
	MaxUsersPerCustomerOrganizationPro       limit.Limit = 10
	MaxUsersPerCustomerOrganizationTrial     limit.Limit = limit.Unlimited

	MaxDeploymentTargetsPerCustomerOrganizationCommunity limit.Limit = 1
	MaxDeploymentTargetsPerCustomerOrganizationStarter   limit.Limit = 1
	MaxDeploymentTargetsPerCustomerOrganizationPro       limit.Limit = 8
	MaxDeploymentTargetsPerCustomerOrganizationTrial                 = limit.Unlimited

	MaxLogExportRowsCommunity limit.Limit = 100
	MaxLogExportRowsStarter   limit.Limit = 100
	MaxLogExportRowsPro       limit.Limit = 10_000
	MaxLogExportRowsTrial     limit.Limit = 10_000

	LogQueryWindowCommunity = 24 * time.Hour
	LogQueryWindowStarter   = 24 * time.Hour
	LogQueryWindowDefault   = 7 * 24 * time.Hour
)

func GetCustomersPerOrganizationLimit(st types.SubscriptionType) limit.Limit {
	switch st {
	case types.SubscriptionTypeCommunity:
		return MaxCustomersPerOrganizationCommunity
	case types.SubscriptionTypeTrial:
		return MaxCustomersPerOrganizationTrial
	case types.SubscriptionTypeStarter:
		return MaxCustomersPerOrganizationStarter
	case types.SubscriptionTypePro:
		return MaxCustomersPerOrganizationPro
	case types.SubscriptionTypeEnterprise:
		return license.GetLicenseData().MaxCustomersPerOrganization
	default:
		panic(fmt.Sprintf("invalid subscription type: %v", st))
	}
}

func GetUsersPerCustomerOrganizationLimit(st types.SubscriptionType) limit.Limit {
	switch st {
	case types.SubscriptionTypeCommunity:
		return MaxUsersPerCustomerOrganizationCommunity
	case types.SubscriptionTypeTrial:
		return MaxUsersPerCustomerOrganizationTrial
	case types.SubscriptionTypeStarter:
		return MaxUsersPerCustomerOrganizationStarter
	case types.SubscriptionTypePro:
		return MaxUsersPerCustomerOrganizationPro
	case types.SubscriptionTypeEnterprise:
		return license.GetLicenseData().MaxUsersPerCustomerOrganization
	default:
		panic(fmt.Sprintf("invalid subscription type: %v", st))
	}
}

func GetDeploymentTargetsPerCustomerOrganizationLimit(st types.SubscriptionType) limit.Limit {
	switch st {
	case types.SubscriptionTypeCommunity:
		return MaxDeploymentTargetsPerCustomerOrganizationCommunity
	case types.SubscriptionTypeTrial:
		return MaxDeploymentTargetsPerCustomerOrganizationTrial
	case types.SubscriptionTypeStarter:
		return MaxDeploymentTargetsPerCustomerOrganizationStarter
	case types.SubscriptionTypePro:
		return MaxDeploymentTargetsPerCustomerOrganizationPro
	case types.SubscriptionTypeEnterprise:
		return license.GetLicenseData().MaxDeploymentTargetsPerCustomerOrganization
	default:
		panic(fmt.Sprintf("invalid subscription type: %v", st))
	}
}

func GetLogExportRowsLimit(st types.SubscriptionType) limit.Limit {
	switch st {
	case types.SubscriptionTypeCommunity:
		return MaxLogExportRowsCommunity
	case types.SubscriptionTypeTrial:
		return MaxLogExportRowsTrial
	case types.SubscriptionTypeStarter:
		return MaxLogExportRowsStarter
	case types.SubscriptionTypePro:
		return MaxLogExportRowsPro
	case types.SubscriptionTypeEnterprise:
		return license.GetLicenseData().MaxLogExportRows
	default:
		panic(fmt.Sprintf("invalid subscription type: %v", st))
	}
}

// GetLogQueryWindow returns how far back log read queries may reach. A planned
// "business" subscription type will extend it up to the full Loki retention period
// (30 days).
func GetLogQueryWindow(st types.SubscriptionType) time.Duration {
	switch st {
	case types.SubscriptionTypeCommunity:
		return LogQueryWindowCommunity
	case types.SubscriptionTypeStarter:
		return LogQueryWindowStarter
	default:
		return LogQueryWindowDefault
	}
}

func GetSubscriptionLimits(st types.SubscriptionType) api.SubscriptionLimits {
	return api.SubscriptionLimits{
		MaxCustomerOrganizations:        GetCustomersPerOrganizationLimit(st).Value(),
		MaxUsersPerCustomerOrganization: GetUsersPerCustomerOrganizationLimit(st).Value(),
		MaxDeploymentsPerCustomerOrg:    GetDeploymentTargetsPerCustomerOrganizationLimit(st).Value(),
		LogQueryWindowSeconds:           int64(GetLogQueryWindow(st) / time.Second),
	}
}
