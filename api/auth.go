package api

import (
	"errors"

	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

type AuthLoginRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	MFACode  *string `json:"mfaCode"`
}

func (r *AuthLoginRequest) Validate() error {
	return validation.ValidateEmail(r.Email)
}

type AuthLoginResponse struct {
	Token       string  `json:"token,omitempty"`
	RequiresMFA bool    `json:"requiresMfa"`
	RedirectURL *string `json:"redirectUrl,omitempty"`
}

type AuthRegistrationRequest struct {
	Name             string `json:"name"`
	OrganizationName string `json:"organizationName"`
	Email            string `json:"email"`
	Password         string `json:"password"`
	TurnstileToken   string `json:"turnstileToken,omitempty"`
}

func (r *AuthRegistrationRequest) Validate() error {
	if err := validation.ValidateEmail(r.Email); err != nil {
		return err
	} else if err := validation.ValidatePassword(r.Password); err != nil {
		return err
	}
	return nil
}

type AuthResetPasswordRequest struct {
	Email string `json:"email"`
}

func (r *AuthResetPasswordRequest) Validate() error {
	return validation.ValidateEmail(r.Email)
}

type AuthResetPasswordConfirmRequest struct {
	Password string  `json:"password"`
	MFACode  *string `json:"mfaCode"`
}

func (r *AuthResetPasswordConfirmRequest) Validate() error {
	return validation.ValidatePassword(r.Password)
}

type AuthSwitchContextRequest struct {
	OrganizationID uuid.UUID `json:"organizationId"`
}

type AuthLensRequest struct {
	CustomerOrganizationID *uuid.UUID `json:"customerOrganizationId,omitempty"`
	PartnerOrganizationID  *uuid.UUID `json:"partnerOrganizationId,omitempty"`
}

func (r *AuthLensRequest) Validate() error {
	if (r.CustomerOrganizationID == nil) == (r.PartnerOrganizationID == nil) {
		return errors.New("either a customer or a partner organization is required")
	}
	return nil
}

type AuthAcceptInviteRequest struct {
	Name     *string `json:"name"`
	Password string  `json:"password"`
	MFACode  *string `json:"mfaCode"`
}

func (r *AuthAcceptInviteRequest) Validate() error {
	return validation.ValidatePassword(r.Password)
}
