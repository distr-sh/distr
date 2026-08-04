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
	// RedirectURL is set when the login happened on the instance's default host while the user's
	// primary organization has a custom app domain: the browser is expected to continue there, so
	// the user ends up on the host that offers their organization's login methods and branding. The
	// token is valid on either host, so an API client can ignore this.
	RedirectURL *string `json:"redirectUrl,omitempty"`
}

type AuthRegistrationRequest struct {
	Name             string `json:"name"`
	OrganizationName string `json:"organizationName"`
	Email            string `json:"email"`
	Password         string `json:"password"`
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

type AuthResetPasswordConfirmRequest struct {
	Password string `json:"password"`
}

func (r *AuthResetPasswordConfirmRequest) Validate() error {
	return validation.ValidatePassword(r.Password)
}

type AuthSwitchContextRequest struct {
	OrganizationID uuid.UUID `json:"organizationId"`
}

type AuthAcceptInviteRequest struct {
	Name     *string `json:"name"`
	Password string  `json:"password"`
}

func (r *AuthAcceptInviteRequest) Validate() error {
	return validation.ValidatePassword(r.Password)
}
