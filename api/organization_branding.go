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

// PortalResponse contains the host-resolved portal branding (browser tab title, favicon and logo) that applies to
// everyone visiting an organization's custom app domain, regardless of authentication. A response is only returned
// when the request host matches a custom app domain, so its presence itself indicates a custom domain.
type PortalResponse struct {
	PageTitle  *string `json:"pageTitle,omitempty"`
	FaviconUrl *string `json:"faviconUrl,omitempty"`
	LogoUrl    *string `json:"logoUrl,omitempty"`
}
