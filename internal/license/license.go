package license

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sync"
	"time"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/licensekey"
	"github.com/distr-sh/distr/internal/limit"
	"github.com/distr-sh/distr/internal/types"
	"github.com/go-viper/mapstructure/v2"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
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

		return jwk.ParseKey(rawPubKey, jwk.WithPEM(true))
	})
)

const licenseDataClaimName = "ld"

var (
	// organizationID, when set at build time via -ldflags, restricts this build to license
	// keys that were issued for the given Distr organization. When empty, organization
	// scoping is disabled and any otherwise valid license key is accepted.
	organizationID string

	// organizationScopeCutoff, when set at build time via -ldflags, is a yyyy-mm-dd date.
	// Only license keys with an "issued at" claim newer than this date are validated
	// against organizationID; keys issued at or before it are exempt for backwards
	// compatibility with license keys minted before organization scoping was introduced.
	organizationScopeCutoff string
)

var cachedOrgScopeCutoff = sync.OnceValues(func() (time.Time, error) {
	if organizationScopeCutoff == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.DateOnly, organizationScopeCutoff)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid organization scope cutoff %q: %w", organizationScopeCutoff, err)
	}
	return t, nil
})

// LicenseData is the parsed private claims from the license key JWT.
type LicenseData struct {
	EnforceLimitsOnStartup bool `mapstructure:"enf"`

	// Global limits
	MaxOrganizations limit.Limit              `mapstructure:"mo"`
	Period           types.SubscriptionPeriod `mapstructure:"p"`

	// Limits for organizations with subscription type Enterprise
	MaxUsersPerOrganization                     limit.Limit `mapstructure:"mou"`
	MaxCustomersPerOrganization                 limit.Limit `mapstructure:"moc"`
	MaxUsersPerCustomerOrganization             limit.Limit `mapstructure:"mcu"`
	MaxDeploymentTargetsPerCustomerOrganization limit.Limit `mapstructure:"mcd"`
	MaxLogExportRows                            limit.Limit `mapstructure:"mlr"`

	ExpirationDate time.Time
}

var (
	cachedLicense      *LicenseData
	defaultLicenseData = LicenseData{
		EnforceLimitsOnStartup:                      false,
		Period:                                      types.SubscriptionPeriodYearly,
		MaxOrganizations:                            limit.Unlimited,
		MaxUsersPerOrganization:                     limit.Unlimited,
		MaxCustomersPerOrganization:                 limit.Unlimited,
		MaxUsersPerCustomerOrganization:             limit.Unlimited,
		MaxDeploymentTargetsPerCustomerOrganization: limit.Unlimited,
		MaxLogExportRows:                            1_000_000,
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

	cutoff, err := cachedOrgScopeCutoff()
	if err != nil {
		return nil, err
	}

	if err := validateOrganizationScope(token, organizationID, cutoff); err != nil {
		return nil, err
	}

	var licenseDataMap map[string]any
	if err := token.Get(licenseDataClaimName, &licenseDataMap); err != nil {
		return nil, fmt.Errorf("invalid license key: %w", err)
	}

	licenseData := defaultLicenseData
	if err := mapstructure.Decode(licenseDataMap, &licenseData); err != nil {
		return nil, fmt.Errorf("invalid license key: %w", err)
	}

	if exp, ok := token.Expiration(); !ok {
		return nil, fmt.Errorf("invalid license key: missing expiration date")
	} else {
		licenseData.ExpirationDate = exp
	}

	return &licenseData, nil
}

// validateOrganizationScope ensures the license key was issued for the organization this build
// is licensed to. It is a noop when no organization ID was configured at build time or when the
// license key was issued at or before the configured cutoff (i.e. before organization scoping
// was introduced).
func validateOrganizationScope(token jwt.Token, expectedOrgID string, cutoff time.Time) error {
	if expectedOrgID == "" {
		return nil
	}

	issuedAt, ok := token.IssuedAt()
	if !ok || !issuedAt.After(cutoff) {
		return nil
	}

	var orgID string
	if err := token.Get(licensekey.OrganizationIDClaimName, &orgID); err != nil {
		return fmt.Errorf("invalid license key: missing organization ID claim")
	} else if orgID != expectedOrgID {
		return fmt.Errorf("invalid license key: organization ID mismatch")
	}

	return nil
}
