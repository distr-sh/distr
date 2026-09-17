/*
Package license encapsulates parsing, validation and accessor logic for the [LicenseData] type.

Explicit initialization is required when using this package. Calling [Initialize] populates the cached [LicenseData]
instance from the Distr license key token provided via the LICENSE_KEY environment variable, given that a public key
for validation is present. The public key must be set at compile time and is embedded with an embed.FS.

A compatible Distr license key can be generated using the following JSON as a template:

	{
		"ld": {
			"enf": true,
			"p": "monthly",
			"t": "enterprise",
			"mo": 123,
			"mou": 123,
			"moc": 123,
			"mcu": 123,
			"mcd": 123,
			"mrs": 123
		}
	}

The "t" claim is the subscription type the instance is licensed for and accepts any of
[types.AllSubscriptionTypes]. It defaults to enterprise, which is the plan every license key
granted before the claim was introduced. When "enf" is true, every organization is set to that
subscription type, is granted its features and uses the limits from the license key rather than
the ones listed for the plan on Distr Cloud.

After error-free initialization, a [LicenseData] object can be obtained via [GetLicenseData].
If no public key is set at compile time, [GetLicenseData] always returns the default values for all limits.

# Organization scoping

A build can additionally be restricted to license keys that were issued for a specific Distr
organization. This is controlled by a variable that is injected at compile time via -ldflags
(similar to the internal/buildconfig variables):

	-X github.com/distr-sh/distr/internal/license.organizationID=<organization uuid>

When organizationID is set, [Initialize] additionally verifies that the license key carries a
matching organization ID claim (see licensekey.OrganizationIDClaimName). When organizationID is
empty, organization scoping is disabled.
*/
package license
