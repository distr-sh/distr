package api

import (
	"time"

	"github.com/google/uuid"
)

type CustomerOrganizationLink struct {
	ID                     uuid.UUID `json:"id"`
	CreatedAt              time.Time `json:"createdAt"`
	CustomerOrganizationID uuid.UUID `json:"customerOrganizationId"`
	Name                   string    `json:"name"`
	Link                   string    `json:"link"`
}

type CreateUpdateCustomerOrganizationLinkRequest struct {
	Name string `json:"name"`
	Link string `json:"link"`
}

type DeleteCustomerOrganizationLinkRequest struct {
	ID uuid.UUID `path:"linkId"`
}
