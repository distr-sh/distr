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
			"mo": 123,
			"mou": 123,
			"moc": 123,
			"mcu": 123,
			"mcd": 123,
			"mlr": 123
		}
	}

After error-free initialization, a [LicenseData] object can be obtained via [GetLicenseData].
If no public key is set at compile time, [GetLicenseData] always returns the default values for all limits.
*/
package license
