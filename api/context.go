package api

import "github.com/distr-sh/distr/internal/types"

type ContextResponse struct {
	User                 UserAccountResponse              `json:"user"`
	Organization         OrganizationResponse             `json:"organization"`
	CustomerOrganization *CustomerOrganization            `json:"customerOrganization,omitempty"`
	SidebarLinks         []SidebarLink                    `json:"sidebarLinks"`
	AvailableContexts    []types.OrganizationWithUserRole `json:"availableContexts,omitempty"`
}
