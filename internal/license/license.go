package license

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"sync"
	"time"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/licensekey"
	"github.com/distr-sh/distr/internal/limit"
	"github.com/distr-sh/distr/internal/types"
	"github.com/go-viper/mapstructure/v2"
	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jwt"
)

var (
	// Using embed.FS allows to handle a missing file at runtime.
	// Should be changed to []byte if we decide that this is a required value.
	//go:embed all:embedded
	efs          embed.FS
	cachedPubKey = sync.OnceValues(func() (jwk.Key, error) {
		f, err := efs.Open("embedded/pubkey.pem")
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, nil
			}

			return nil, err
		}
		defer f.Close()

		rawPubKey, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}

		return jwk.ParseKey(rawPubKey, jwk.WithX509(true))
	})
)

const licenseDataClaimName = "ld"

// organizationID, when set at build time via -ldflags, restricts this build to license
// keys that were issued for the given Distr organization. When empty, organization
// scoping is disabled and any otherwise valid license key is accepted.
var organizationID string

// LicenseData is the parsed private claims from the license key JWT.
type LicenseData struct {
	EnforceLimitsOnStartup bool `mapstructure:"enf"`

	// Global limits
	MaxOrganizations limit.Limit              `mapstructure:"mo"`
	Period           types.SubscriptionPeriod `mapstructure:"p"`
	SubscriptionType types.SubscriptionType   `mapstructure:"t"`

	// Limits of every organization on an instance that enforces the limits of its license key
	MaxUsersPerOrganization                     limit.Limit `mapstructure:"mou"`
	MaxCustomersPerOrganization                 limit.Limit `mapstructure:"moc"`
	MaxUsersPerCustomerOrganization             limit.Limit `mapstructure:"mcu"`
	MaxDeploymentTargetsPerCustomerOrganization limit.Limit `mapstructure:"mcd"`
	MaxRegistryStorageBytes                     limit.Limit `mapstructure:"mrs"`

	ExpirationDate time.Time
}

// Plan is the subscription every organization of the instance is on when the license key
// enforces its limits.
func (ld LicenseData) Plan() types.SubscriptionPlan {
	return types.SubscriptionPlan{
		Type:                    ld.SubscriptionType,
		Period:                  ld.Period,
		EndsAt:                  ld.ExpirationDate,
		CustomerOrganizationQty: ld.MaxCustomersPerOrganization,
		UserAccountQty:          ld.MaxUsersPerOrganization,
	}
}

var (
	cachedLicense      *LicenseData
	defaultLicenseData = LicenseData{
		EnforceLimitsOnStartup:                      false,
		Period:                                      types.SubscriptionPeriodYearly,
		SubscriptionType:                            types.SubscriptionTypeEnterprise,
		MaxOrganizations:                            limit.Unlimited,
		MaxUsersPerOrganization:                     limit.Unlimited,
		MaxCustomersPerOrganization:                 limit.Unlimited,
		MaxUsersPerCustomerOrganization:             limit.Unlimited,
		MaxDeploymentTargetsPerCustomerOrganization: limit.Unlimited,
		MaxRegistryStorageBytes:                     limit.Unlimited,
	}
)

func Initialize() error {
	if parsed, err := parseAndValidate(cachedPubKey, env.LicenseKey()); err != nil {
		return fmt.Errorf("license key initialization: %w", err)
	} else {
		cachedLicense = parsed
	}

	return nil
}

// GetLicenseData MUST be called after [Initialize], otherwise it WILL panic.
func GetLicenseData() LicenseData {
	if cachedLicense == nil {
		panic("detected call to license.GetLicenseData before calling license.Initialize")
	}

	return *cachedLicense
}

func parseAndValidate(pubKeySrc func() (jwk.Key, error), licenseKey string) (*LicenseData, error) {
	key, err := pubKeySrc()
	if err != nil {
		return nil, fmt.Errorf("read validation key: %w", err)
	} else if key == nil {
		return &defaultLicenseData, nil
	} else if licenseKey == "" {
		return nil, errors.New("distr license key is required via environment variable LICENSE_KEY")
	}

	token, err := jwt.ParseString(licenseKey, jwt.WithKey(jwa.EdDSA(), key))
	if err != nil {
		return nil, fmt.Errorf("invalid license key: %w", err)
	}

	if err := validateOrganizationScope(token, organizationID); err != nil {
		return nil, err
	}

	licenseDataMap, err := jwt.Get[map[string]any](token, licenseDataClaimName)
	if err != nil {
		return nil, fmt.Errorf("invalid license key: %w", err)
	}

	licenseData := defaultLicenseData
	if err := mapstructure.Decode(licenseDataMap, &licenseData); err != nil {
		return nil, fmt.Errorf("invalid license key: %w", err)
	}

	if !slices.Contains(types.AllSubscriptionTypes(), licenseData.SubscriptionType) {
		return nil, fmt.Errorf("invalid license key: unknown subscription type %q", licenseData.SubscriptionType)
	}

	if exp, ok := token.Expiration(); !ok {
		return nil, fmt.Errorf("invalid license key: missing expiration date")
	} else {
		licenseData.ExpirationDate = exp
	}

	return &licenseData, nil
}

// validateOrganizationScope ensures the license key was issued for the organization this build
// is licensed to. It is a noop when no organization ID was configured at build time.
func validateOrganizationScope(token jwt.Token, expectedOrgID string) error {
	if expectedOrgID == "" {
		return nil
	}

	if orgID, err := jwt.Get[string](token, licensekey.OrganizationIDClaimName); err != nil {
		return fmt.Errorf("invalid license key: missing organization ID claim")
	} else if orgID != expectedOrgID {
		return fmt.Errorf("invalid license key: organization ID mismatch")
	}

	return nil
}
