package api

import (
	"github.com/google/uuid"
)

type UpsertOrganizationBrandingRequest struct {
	Title *string `json:"title"`
	// The description is rendered as markdown, where leading whitespace is syntax.
	Description    *string    `json:"description" trim:"-"`
	LogoImageID    *uuid.UUID `json:"logoImageId"`
	PageTitle      *string    `json:"pageTitle"`
	FaviconImageID *uuid.UUID `json:"faviconImageId"`
}
