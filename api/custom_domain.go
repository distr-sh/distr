package api

import (
	"fmt"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
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

func (r *CreateCustomDomainRequest) Validate() error {
	if r.DomainType != types.DomainTypeApp && r.DomainType != types.DomainTypeRegistry {
		return validation.NewValidationFailedError(
			fmt.Sprintf("domainType must be %q or %q", types.DomainTypeApp, types.DomainTypeRegistry),
		)
	}
	return validation.ValidateHostname(r.Domain)
}

type CreateCustomDomainsRequest struct {
	Domains []CreateCustomDomainRequest `json:"domains"`
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
