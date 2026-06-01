package api

import (
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

type CreateUserAccountRequest struct {
	Email                  string         `json:"email"`
	Name                   string         `json:"name"`
	UserRole               types.UserRole `json:"userRole"`
	CustomerOrganizationID *uuid.UUID     `json:"customerOrganizationId,omitempty"`
	PartnerOrganizationID  *uuid.UUID     `json:"partnerOrganizationId,omitempty"`
}

type CreateUserAccountResponse struct {
	User      types.UserAccountWithUserRole `json:"user"`
	InviteURL string                        `json:"inviteUrl"`
}

type UserAccountResponse struct {
	types.UserAccountWithUserRole
	ImageUrl *string `json:"imageUrl,omitempty"`
}

type UpdateUserAccountRequest struct {
	Name     *string    `json:"name"`
	Password *string    `json:"password"`
	ImageID  *uuid.UUID `json:"imageId"`
}

func (r UpdateUserAccountRequest) Validate() error {
	if r.Password != nil {
		if err := validation.ValidatePassword(*r.Password); err != nil {
			return err
		}
	}
	return nil
}

type UpdateUserAccountEmailRequest struct {
	Email string `json:"email"`
}

func (r UpdateUserAccountEmailRequest) Validate() error {
	if err := validation.ValidateEmail(r.Email); err != nil {
		return err
	}
	return nil
}

type PatchUserAccountRequest struct {
	Name     *string         `json:"name"`
	UserRole *types.UserRole `json:"userRole"`
}
