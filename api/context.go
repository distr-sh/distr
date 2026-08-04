package api

import "github.com/distr-sh/distr/internal/types"

type ContextResponse struct {
	User                 UserAccountResponse              `json:"user"`
	Organization         OrganizationResponse             `json:"organization"`
	CustomerOrganization *CustomerOrganization            `json:"customerOrganization,omitempty"`
	PartnerOrganization  *PartnerOrganization             `json:"partnerOrganization,omitempty"`
	SidebarLinks         []SidebarLink                    `json:"sidebarLinks,omitempty"`
	AvailableContexts    []types.OrganizationWithUserRole `json:"availableContexts,omitempty"`
	// RegistryHost is the effective registry host of the organization, considering custom
	// domains and legacy branding domains, falling back to the instance default.
	RegistryHost string `json:"registryHost,omitzero"`
	// CanCreateOrganization reports whether the user may create another organization. It is false
	// for a user who signs in through an organization's own identity provider: such an account has
	// to stay a member of that one organization, so creating another one is refused. Derived from
	// the account's current state rather than from the token, so it cannot go stale.
	CanCreateOrganization bool `json:"canCreateOrganization"`
}
