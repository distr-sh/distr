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
	ID        uuid.UUID  `db:"id"          json:"id"`
	CreatedAt time.Time  `db:"created_at"  json:"createdAt"`
	Domain    string     `db:"domain"      json:"domain"`
	Type      DomainType `db:"domain_type" json:"domainType"`
	// OrganizationID references the vendor organization that owns the domain.
	OrganizationID uuid.UUID `db:"organization_id" json:"organizationId"`
	// CustomerOrganizationID is set on a customer_portal domain that belongs to one customer. When it is
	// nil, a customer_portal domain is the vendor's shared portal for all of its customers.
	CustomerOrganizationID *uuid.UUID `db:"customer_organization_id" json:"customerOrganizationId,omitempty"`
}
