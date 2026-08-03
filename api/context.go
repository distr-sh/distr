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
}
