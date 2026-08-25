package api

import (
	"fmt"
	"slices"
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

type CustomDomain struct {
	ID         uuid.UUID        `json:"id"`
	CreatedAt  time.Time        `json:"createdAt"`
	Domain     string           `json:"domain"`
	DomainType types.DomainType `json:"domainType"`
	// OrganizationID references the vendor organization that owns the domain.
	OrganizationID uuid.UUID `json:"organizationId"`
	// CustomerOrganizationID is set on a customer_portal domain that belongs to one customer. When it is
	// nil, a customer_portal domain is the vendor's shared portal for all of its customers.
	CustomerOrganizationID *uuid.UUID `json:"customerOrganizationId,omitempty"`
	// Verified reports whether the domain is currently used for links, mails, agent manifests and
	// registry URLs. It is decided by the server so that no client has to restate the rule.
	Verified              bool       `json:"verified"`
	VerifiedAt            *time.Time `json:"verifiedAt,omitempty"`
	VerificationCheckedAt *time.Time `json:"verificationCheckedAt,omitempty"`
	// VerificationError is why the domain's CNAME record was last found not to point at this instance.
	VerificationError *string `json:"verificationError,omitempty"`
}

type CreateCustomDomainRequest struct {
	Domain     string           `json:"domain"`
	DomainType types.DomainType `json:"domainType"`
}

// Normalize reduces the domain to the bare hostname it is validated and stored as, so that a
// value pasted as a URL is accepted as well.
func (r *CreateCustomDomainRequest) Normalize() {
	r.Domain = validation.NormalizeHostname(r.Domain)
}

var customDomainTypes = []types.DomainType{
	types.DomainTypeApp,
	types.DomainTypeRegistry,
	types.DomainTypeCustomerPortal,
}

func (r *CreateCustomDomainRequest) Validate() error {
	if !slices.Contains(customDomainTypes, r.DomainType) {
		return validation.NewValidationFailedError(
			fmt.Sprintf("domainType must be one of %v", customDomainTypes),
		)
	}
	return validation.ValidateHostname(r.Domain)
}

type CreateCustomDomainsRequest struct {
	Domains []CreateCustomDomainRequest `json:"domains"`
	// CustomerOrganizationID targets a customer's own domain instead of the caller's organization.
	// Only a vendor or partner admin may set it; a customer caller may only ever target itself.
	CustomerOrganizationID *uuid.UUID `json:"customerOrganizationId,omitempty"`
}

func (r *CreateCustomDomainsRequest) Normalize() {
	for i := range r.Domains {
		r.Domains[i].Normalize()
	}
}

func (r *CreateCustomDomainsRequest) Validate() error {
	if len(r.Domains) == 0 {
		return validation.NewValidationFailedError("at least one domain is required")
	}
	for _, domain := range r.Domains {
		if err := domain.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// CustomDomainVerification is the stored outcome of the last CNAME check of one domain. Inconclusive
// reports that the last check did not complete, in which case the domain keeps the state it had
// before: VerificationCheckedAt is the inconclusive attempt, while VerifiedAt and VerificationError
// still describe the last check that reached a conclusion.
type CustomDomainVerification struct {
	CustomDomainID        uuid.UUID  `json:"customDomainId"`
	Verified              bool       `json:"verified"`
	VerifiedAt            *time.Time `json:"verifiedAt,omitempty"`
	VerificationCheckedAt *time.Time `json:"verificationCheckedAt,omitempty"`
	VerificationError     *string    `json:"verificationError,omitempty"`
	Inconclusive          bool       `json:"inconclusive"`
}
