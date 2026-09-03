package types

import (
	"time"

	"github.com/google/uuid"
)

type DomainType string

const (
	DomainTypeApp            DomainType = "app"
	DomainTypeRegistry       DomainType = "registry"
	DomainTypeCustomerPortal DomainType = "customer_portal"
)

type CustomDomain struct {
	ID        uuid.UUID  `db:"id"`
	CreatedAt time.Time  `db:"created_at"`
	Domain    string     `db:"domain"`
	Type      DomainType `db:"domain_type"`
	// OrganizationID references the vendor organization that owns the domain.
	OrganizationID uuid.UUID `db:"organization_id"`
	// CustomerOrganizationID is set on a customer_portal domain that belongs to one customer. When it is
	// nil, a customer_portal domain is the vendor's shared portal for all of its customers.
	CustomerOrganizationID *uuid.UUID `db:"customer_organization_id"`
	// VerifiedAt is the last time the domain's CNAME record was found to point at this instance. It is
	// never cleared, so a domain that fails a later check keeps it.
	VerifiedAt *time.Time `db:"verified_at"`
	// VerificationCheckedAt is the last completed check, whatever its outcome.
	VerificationCheckedAt *time.Time `db:"verification_checked_at"`
	// VerificationError is the reason the last check found the record to be wrong, and nil when it
	// succeeded. A lookup that did not complete says nothing about the record and leaves it as it was.
	VerificationError *string `db:"verification_error"`
}

// Verified reports whether the domain may be used for links, mails, agent manifests and registry
// URLs. It is deliberately not a freshness check on VerifiedAt: a domain is dropped on evidence that
// its record is wrong, never because verification merely stopped happening.
func (d CustomDomain) Verified() bool {
	return d.VerifiedAt != nil && d.VerificationError == nil
}
