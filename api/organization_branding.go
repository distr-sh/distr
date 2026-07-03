package api

import (
	"github.com/google/uuid"
)

type CreateOrUpdateOrganizationBrandingRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	LogoImageID *uuid.UUID `json:"logoImageId"`
}
