package types

type License struct {
	CustomerOrganization    CustomerOrganization     `json:"customerOrganization"`
	ApplicationEntitlements []ApplicationEntitlement `json:"applicationEntitlements"`
	ArtifactEntitlements    []ArtifactEntitlement    `json:"artifactEntitlements"`
	LicenseKeys             []LicenseKey             `json:"licenseKeys"`
}
