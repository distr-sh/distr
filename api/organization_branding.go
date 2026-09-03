package api

import (
	"github.com/google/uuid"
)

type UpsertOrganizationBrandingRequest struct {
	Title          *string    `json:"title"`
	Description    *string    `json:"description"`
	LogoImageID    *uuid.UUID `json:"logoImageId"`
	PageTitle      *string    `json:"pageTitle"`
	FaviconImageID *uuid.UUID `json:"faviconImageId"`
}
