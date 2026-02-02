package api

import (
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

type AuthLoginRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	MFACode  *string `json:"mfaCode"`
}

type AuthLoginResponse struct {
	Token       string `json:"token,omitempty"`
	RequiresMFA bool   `json:"requiresMfa"`
}

type AuthRegistrationRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *AuthRegistrationRequest) Validate() error {
	if r.Email == "" {
		return validation.NewValidationFailedError("email is empty")
	} else if err := validation.ValidatePassword(r.Password); err != nil {
		return err
	}
	return nil
}

type AuthResetPasswordRequest struct {
	Email string `json:"email"`
}

func (r *AuthResetPasswordRequest) Validate() error {
	if r.Email == "" {
		return validation.NewValidationFailedError("email is empty")
	}
	return nil
}

type AuthSwitchContextRequest struct {
	OrganizationID uuid.UUID `json:"organizationId"`
}
