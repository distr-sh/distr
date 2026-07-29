package types

import (
	"time"

	"github.com/google/uuid"
)

type DomainType string

const (
	DomainTypeApp      DomainType = "app"
	DomainTypeRegistry DomainType = "registry"
)

type CustomDomain struct {
	ID        uuid.UUID  `db:"id"          json:"id"`
	CreatedAt time.Time  `db:"created_at"  json:"createdAt"`
	Domain    string     `db:"domain"      json:"domain"`
	Type      DomainType `db:"domain_type" json:"domainType"`
	// OrganizationID references the vendor organization that owns the domain.
	OrganizationID uuid.UUID `db:"organization_id" json:"organizationId"`
	// CustomerOrganizationID and PartnerOrganizationID optionally narrow the domain
	// to a single customer or partner organization. Both unset means the domain is
	// the org-wide domain. No API surface for them yet (DEV-593).
	CustomerOrganizationID *uuid.UUID `db:"customer_organization_id" json:"customerOrganizationId,omitempty"`
	PartnerOrganizationID  *uuid.UUID `db:"partner_organization_id"  json:"partnerOrganizationId,omitempty"`
}
