package api

import "github.com/distr-sh/distr/internal/types"

type CustomerOrganizationResponse struct {
	CustomerOrganization
	Links []CustomerOrganizationLink `json:"links"`
}

type ContextResponse struct {
	User                 UserAccountResponse              `json:"user"`
	Organization         OrganizationResponse             `json:"organization"`
	CustomerOrganization *CustomerOrganizationResponse    `json:"customerOrganization,omitempty"`
	AvailableContexts    []types.OrganizationWithUserRole `json:"availableContexts,omitempty"`
}
