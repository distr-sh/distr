package api

import (
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

type ControllerImageRegistry struct {
	ImageRegistry        *string `json:"imageRegistry,omitempty"`
	DefaultImageRegistry string  `json:"defaultImageRegistry"`
}

type UpdateControllerImageRegistryRequest struct {
	ImageRegistry          string     `json:"imageRegistry"`
	CustomerOrganizationID *uuid.UUID `json:"customerOrganizationId,omitempty"`
}

func (r *UpdateControllerImageRegistryRequest) Normalize() {
	r.ImageRegistry = validation.NormalizeImageRegistry(r.ImageRegistry)
}

func (r *UpdateControllerImageRegistryRequest) Validate() error {
	if r.ImageRegistry == "" {
		return nil
	}
	return validation.ValidateImageRegistry(r.ImageRegistry)
}
