package license

import (
	"errors"
	"fmt"
	"sync"

	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/limit"
	"github.com/go-viper/mapstructure/v2"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type LicenseData struct {
	EnforceLimitsOnStartup bool `mapstructure:"enforce_limits_on_startup"`

	// Global limits

	MaxOrganizations limit.Limit `mapstructure:"max_organizations"`

	// Limits for organizations with subscription type Enterprise

	MaxUsersPerOrganization                     limit.Limit `mapstructure:"max_users_per_organization"`
	MaxCustomersPerOrganization                 limit.Limit `mapstructure:"max_customers_per_organization"`
	MaxUsersPerCustomerOrganization             limit.Limit `mapstructure:"max_users_per_customer_organization"`
	MaxDeploymentTargetsPerCustomerOrganization limit.Limit `mapstructure:"max_deployment_targets_per_customer_organization"` //nolint:lll
	MaxLogExportRows                            limit.Limit `mapstructure:"max_log_export_rows"`
}

var defaultLicenseData = LicenseData{
	MaxOrganizations:                            limit.Unlimited,
	MaxUsersPerOrganization:                     limit.Unlimited,
	MaxCustomersPerOrganization:                 limit.Unlimited,
	MaxUsersPerCustomerOrganization:             limit.Unlimited,
	MaxDeploymentTargetsPerCustomerOrganization: limit.Unlimited,
	MaxLogExportRows:                            1_000_000,
}

var (
	cachedJwkSet = sync.OnceValues(func() (jwk.Set, error) {
		if key := buildconfig.LicenseValidationPublicKey(); key == "" {
			return nil, nil
		} else {
			return jwk.ParseString(key)
		}
	})
	cachedLicenseData *LicenseData
)

func Initialize() error {
	if licenseData, err := parseAndValidate(env.LicenseKey()); err != nil {
		return fmt.Errorf("invalid license key: %w", err)
	} else {
		cachedLicenseData = licenseData
	}

	return nil
}

// GetLicenseData MUST be called after [Initialize], otherwise it WILL panic.
func GetLicenseData() LicenseData {
	if cachedLicenseData == nil {
		// panic with a more useful error message than "nil pointer dereference"
		panic("detected call to license.GetLicenseData before calling license.Initialize")
	}

	return *cachedLicenseData
}

func parseAndValidate(licenseKey string) (*LicenseData, error) {
	jwkSet, err := cachedJwkSet()
	if err != nil {
		return nil, fmt.Errorf("validate license key: %w", err)
	} else if jwkSet == nil {
		return &defaultLicenseData, nil
	} else if licenseKey == "" {
		return nil, errors.New("license key is required")
	}

	jwt, err := jwt.ParseString(licenseKey, jwt.WithKeySet(jwkSet))
	if err != nil {
		return nil, fmt.Errorf("invalid license key: %w", err)
	}

	var licenseData LicenseData
	if err := mapstructure.Decode(jwt.PrivateClaims(), &licenseData); err != nil {
		return nil, fmt.Errorf("invalid license key: %w", err)
	}
	return &licenseData, nil
}
