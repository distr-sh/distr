package api

import "github.com/distr-sh/distr/internal/validation"

type CustomerSettingsResponse struct {
	ControllerImageRepository        *string `json:"controllerImageRepository,omitempty"`
	DefaultControllerImageRepository string  `json:"defaultControllerImageRepository"`
}

type UpdateCustomerSettingsRequest struct {
	ControllerImageRepository *string `json:"controllerImageRepository"`
}

func (r *UpdateCustomerSettingsRequest) Normalize() {
	if r.ControllerImageRepository == nil {
		return
	}
	if value := validation.NormalizeImageRepository(*r.ControllerImageRepository); value != "" {
		r.ControllerImageRepository = &value
	} else {
		r.ControllerImageRepository = nil
	}
}

func (r *UpdateCustomerSettingsRequest) Validate() error {
	if r.ControllerImageRepository == nil {
		return nil
	}
	return validation.ValidateImageRepository(*r.ControllerImageRepository)
}
