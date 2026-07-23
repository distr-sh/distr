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
	MaxCustomersPerOrganizationPro       limit.Limit = 100
	MaxCustomersPerOrganizationBusiness              = limit.Unlimited
	MaxCustomersPerOrganizationTrial                 = limit.Unlimited

	MaxUsersPerCustomerOrganizationCommunity limit.Limit = 1
	MaxUsersPerCustomerOrganizationPro       limit.Limit = 10
	MaxUsersPerCustomerOrganizationBusiness  limit.Limit = 25
	MaxUsersPerCustomerOrganizationTrial     limit.Limit = limit.Unlimited

	MaxDeploymentTargetsPerCustomerOrganizationCommunity limit.Limit = 1
	MaxDeploymentTargetsPerCustomerOrganizationPro       limit.Limit = 8
	MaxDeploymentTargetsPerCustomerOrganizationBusiness  limit.Limit = 8
	MaxDeploymentTargetsPerCustomerOrganizationTrial                 = limit.Unlimited

	MaxLogExportRowsCommunity limit.Limit = 100
	MaxLogExportRowsPro       limit.Limit = 10_000
	MaxLogExportRowsBusiness  limit.Limit = 10_000
	MaxLogExportRowsTrial     limit.Limit = 10_000

	LogQueryWindowCommunity = 24 * time.Hour
	LogQueryWindowBusiness  = 30 * 24 * time.Hour
	LogQueryWindowDefault   = 7 * 24 * time.Hour
)

func GetCustomersPerOrganizationLimit(st types.SubscriptionType) limit.Limit {
	switch st {
	case types.SubscriptionTypeCommunity:
		return MaxCustomersPerOrganizationCommunity
	case types.SubscriptionTypeTrial:
		return MaxCustomersPerOrganizationTrial
	case types.SubscriptionTypePro:
		return MaxCustomersPerOrganizationPro
	case types.SubscriptionTypeBusiness:
		return MaxCustomersPerOrganizationBusiness
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
	case types.SubscriptionTypePro:
		return MaxUsersPerCustomerOrganizationPro
	case types.SubscriptionTypeBusiness:
		return MaxUsersPerCustomerOrganizationBusiness
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
	case types.SubscriptionTypePro:
		return MaxDeploymentTargetsPerCustomerOrganizationPro
	case types.SubscriptionTypeBusiness:
		return MaxDeploymentTargetsPerCustomerOrganizationBusiness
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
	case types.SubscriptionTypePro:
		return MaxLogExportRowsPro
	case types.SubscriptionTypeBusiness:
		return MaxLogExportRowsBusiness
	case types.SubscriptionTypeEnterprise:
		return license.GetLicenseData().MaxLogExportRows
	default:
		panic(fmt.Sprintf("invalid subscription type: %v", st))
	}
}

// GetLogQueryWindow returns how far back log read queries may reach.
// The business subscription type extends it up to the full Loki retention period (30 days).
func GetLogQueryWindow(st types.SubscriptionType) time.Duration {
	switch st {
	case types.SubscriptionTypeCommunity:
		return LogQueryWindowCommunity
	case types.SubscriptionTypeBusiness:
		return LogQueryWindowBusiness
	default:
		return LogQueryWindowDefault
	}
}

// LogQueryWindowTimezoneSlack is the extra period accepted for explicitly requested
// log query start timestamps, on top of the exact subscription window. The frontend
// limits the range picker to 00:00 local time of the first day inside the window,
// whose UTC instant is unknown to the server but always within 24 hours before the
// exact window boundary. It must not be applied to default (unset) query starts.
const LogQueryWindowTimezoneSlack = 24 * time.Hour

// GetLogQueryWindowStart returns the default start for log read queries,
// i.e. the start of the exact subscription window.
func GetLogQueryWindowStart(st types.SubscriptionType) time.Time {
	return time.Now().Add(-GetLogQueryWindow(st))
}

func GetSubscriptionLimits(st types.SubscriptionType) api.SubscriptionLimits {
	return api.SubscriptionLimits{
		MaxCustomerOrganizations:        GetCustomersPerOrganizationLimit(st).Value(),
		MaxUsersPerCustomerOrganization: GetUsersPerCustomerOrganizationLimit(st).Value(),
		MaxDeploymentsPerCustomerOrg:    GetDeploymentTargetsPerCustomerOrganizationLimit(st).Value(),
		LogQueryWindowSeconds:           int64(GetLogQueryWindow(st) / time.Second),
	}
}
