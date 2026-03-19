package license

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sync"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/limit"
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

// LicenseData is the parsed private claims from the license key JWT.
type LicenseData struct {
	EnforceLimitsOnStartup bool `mapstructure:"enf"`

	// Global limits

	MaxOrganizations limit.Limit `mapstructure:"mo"`

	// Limits for organizations with subscription type Enterprise

	MaxUsersPerOrganization                     limit.Limit `mapstructure:"mou"`
	MaxCustomersPerOrganization                 limit.Limit `mapstructure:"moc"`
	MaxUsersPerCustomerOrganization             limit.Limit `mapstructure:"mcu"`
	MaxDeploymentTargetsPerCustomerOrganization limit.Limit `mapstructure:"mcd"`
	MaxLogExportRows                            limit.Limit `mapstructure:"mlr"`
}

var (
	cachedLicenseData  *LicenseData
	defaultLicenseData = LicenseData{
		EnforceLimitsOnStartup:                      false,
		MaxOrganizations:                            limit.Unlimited,
		MaxUsersPerOrganization:                     limit.Unlimited,
		MaxCustomersPerOrganization:                 limit.Unlimited,
		MaxUsersPerCustomerOrganization:             limit.Unlimited,
		MaxDeploymentTargetsPerCustomerOrganization: limit.Unlimited,
		MaxLogExportRows:                            1_000_000,
	}
)

func Initialize() error {
	if licenseData, err := parseAndValidate(cachedPubKey, env.LicenseKey()); err != nil {
		return fmt.Errorf("license key initialization: %w", err)
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

func parseAndValidate(pubKeySrc func() (jwk.Key, error), licenseKey string) (*LicenseData, error) {
	key, err := pubKeySrc()
	if err != nil {
		return nil, fmt.Errorf("read validation key: %w", err)
	} else if key == nil {
		return &defaultLicenseData, nil
	} else if licenseKey == "" {
		return nil, errors.New("license key is required")
	}

	token, err := jwt.ParseString(licenseKey, jwt.WithKey(jwa.EdDSA(), key))
	if err != nil {
		return nil, fmt.Errorf("invalid license key: %w", err)
	}

	var licenseDataMap map[string]any
	if err := token.Get(licenseDataClaimName, &licenseDataMap); err != nil {
		return nil, fmt.Errorf("invalid license key: %w", err)
	}

	licenseData := defaultLicenseData
	if err := mapstructure.Decode(licenseDataMap, &licenseData); err != nil {
		return nil, fmt.Errorf("invalid license key: %w", err)
	}

	return &licenseData, nil
}
