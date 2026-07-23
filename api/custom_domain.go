package api

import (
	"fmt"
	"strings"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
)

type CreateCustomDomainRequest struct {
	Domain     string           `json:"domain"`
	DomainType types.DomainType `json:"domainType"`
}

// Normalize lower-cases the domain and strips surrounding whitespace and a trailing dot,
// so it can be validated and stored as a bare hostname.
func (r *CreateCustomDomainRequest) Normalize() {
	r.Domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(r.Domain)), ".")
}

func (r *CreateCustomDomainRequest) Validate() error {
	if r.DomainType != types.DomainTypeApp && r.DomainType != types.DomainTypeRegistry {
		return validation.NewValidationFailedError(
			fmt.Sprintf("domainType must be %q or %q", types.DomainTypeApp, types.DomainTypeRegistry),
		)
	}
	return validation.ValidateHostname(r.Domain)
}
