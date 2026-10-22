package api

import (
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

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

type ContextSwitchTargetsResponse struct {
	Organizations []ContextSwitchOrganization `json:"organizations"`
}

type ContextSwitchOrganization struct {
	ID                    uuid.UUID             `json:"id"`
	LogoURL               *string               `json:"logoUrl,omitempty"`
	CustomerOrganizations []ContextSwitchPortal `json:"customerOrganizations"`
	PartnerOrganizations  []ContextSwitchPortal `json:"partnerOrganizations"`
}

type ContextSwitchPortal struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	ImageURL *string   `json:"imageUrl,omitempty"`
}
