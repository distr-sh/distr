package types

import (
	"slices"
	"time"

	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
)

type UserAccount struct {
	ID                     uuid.UUID  `db:"id" json:"id"`
	CreatedAt              time.Time  `db:"created_at" json:"createdAt"`
	Email                  string     `db:"email" json:"email"`
	EmailVerifiedAt        *time.Time `db:"email_verified_at" json:"-"`
	EmailVerified          bool       `db:"email_verified" json:"emailVerified"`
	PasswordHash           []byte     `db:"password_hash" json:"-"`
	PasswordSalt           []byte     `db:"password_salt" json:"-"`
	Name                   string     `db:"name" json:"name,omitempty"`
	ImageID                *uuid.UUID `db:"image_id" json:"-"`
	LastUsedOrganizationID *uuid.UUID `db:"last_used_organization_id" json:"-"`
	MFASecret              *string    `db:"mfa_secret" json:"-"`
	MFAEnabled             bool       `db:"mfa_enabled" json:"mfaEnabled"`
	MFAEnabledAt           *time.Time `db:"mfa_enabled_at" json:"-"`
	IsSuperAdmin           bool       `db:"is_super_admin" json:"-"`
	Password               string     `db:"-" json:"-"`
	// Remember to update AsUserAccountWithRole when adding fields!
}

func (u *UserAccount) AsUserAccountWithRole(
	role UserRole,
	customerOrganizationID *uuid.UUID,
	joinedOrgAt time.Time,
) UserAccountWithUserRole {
	return UserAccountWithUserRole{
		ID:                     u.ID,
		CreatedAt:              u.CreatedAt,
		Email:                  u.Email,
		EmailVerifiedAt:        util.PtrCopy(u.EmailVerifiedAt),
		PasswordHash:           slices.Clone(u.PasswordHash),
		PasswordSalt:           slices.Clone(u.PasswordSalt),
		Name:                   u.Name,
		ImageID:                u.ImageID,
		MFASecret:              util.PtrCopy(u.MFASecret),
		MFAEnabled:             u.MFAEnabled,
		MFAEnabledAt:           util.PtrCopy(u.MFAEnabledAt),
		IsSuperAdmin:           u.IsSuperAdmin,
		Password:               u.Password,
		UserRole:               role,
		JoinedOrgAt:            joinedOrgAt,
		CustomerOrganizationID: customerOrganizationID,
		LastUsedOrganizationID: u.LastUsedOrganizationID,
	}
}

type UserAccountWithUserRole struct {
	// copy+pasted from UserAccount because pgx does not like embedded structs
	ID                     uuid.UUID  `db:"id" json:"id"`
	CreatedAt              time.Time  `db:"created_at" json:"createdAt"`
	Email                  string     `db:"email" json:"email"`
	EmailVerifiedAt        *time.Time `db:"email_verified_at" json:"-"`
	EmailVerified          bool       `db:"email_verified" json:"emailVerified"`
	PasswordHash           []byte     `db:"password_hash" json:"-"`
	PasswordSalt           []byte     `db:"password_salt" json:"-"`
	Name                   string     `db:"name" json:"name,omitempty"`
	ImageID                *uuid.UUID `db:"image_id" json:"imageId,omitempty"`
	LastUsedOrganizationID *uuid.UUID `db:"last_used_organization_id" json:"-"`
	MFASecret              *string    `db:"mfa_secret" json:"-"`
	MFAEnabled             bool       `db:"mfa_enabled" json:"mfaEnabled"`
	MFAEnabledAt           *time.Time `db:"mfa_enabled_at" json:"-"`
	IsSuperAdmin           bool       `db:"is_super_admin" json:"-"`
	// not copy+pasted
	UserRole UserRole `db:"user_role" json:"userRole"`
	// not copy+pasted
	JoinedOrgAt time.Time `db:"joined_org_at" json:"joinedOrgAt"`
	// not copy+pasted
	CustomerOrganizationID *uuid.UUID `db:"customer_organization_id" json:"customerOrganizationId,omitempty"`
	Password               string     `db:"-" json:"-"`
	// Remember to update AsUserAccount when adding fields!
}

func (u *UserAccountWithUserRole) AsUserAccount() UserAccount {
	return UserAccount{
		ID:                     u.ID,
		CreatedAt:              u.CreatedAt,
		Email:                  u.Email,
		EmailVerifiedAt:        util.PtrCopy(u.EmailVerifiedAt),
		PasswordHash:           slices.Clone(u.PasswordHash),
		PasswordSalt:           slices.Clone(u.PasswordSalt),
		Name:                   u.Name,
		ImageID:                u.ImageID,
		LastUsedOrganizationID: u.LastUsedOrganizationID,
		MFASecret:              util.PtrCopy(u.MFASecret),
		MFAEnabled:             u.MFAEnabled,
		MFAEnabledAt:           util.PtrCopy(u.MFAEnabledAt),
		IsSuperAdmin:           u.IsSuperAdmin,
		Password:               u.Password,
	}
}
