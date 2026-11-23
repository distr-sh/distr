package types

// OrganizationKind is whom a member of a vendor's organization belongs to: the vendor itself, one of its partners
// or one of its customers.
type OrganizationKind string

const (
	OrganizationKindVendor   OrganizationKind = "vendor"
	OrganizationKindPartner  OrganizationKind = "partner"
	OrganizationKindCustomer OrganizationKind = "customer"
)
