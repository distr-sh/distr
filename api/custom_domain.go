package api

import (
	"fmt"
	"slices"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

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
