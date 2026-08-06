package api

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

const customOIDCConfigurationNameMaxLength = 100

type CustomOIDCConfiguration struct {
	ID                  uuid.UUID      `json:"id"`
	CreatedAt           time.Time      `json:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt"`
	CustomDomainID      uuid.UUID      `json:"customDomainId"`
	Name                string         `json:"name"`
	Slug                string         `json:"slug"`
	Enabled             bool           `json:"enabled"`
	Issuer              string         `json:"issuer"`
	ClientID            string         `json:"clientId"`
	ClientSecretSet     bool           `json:"clientSecretSet"`
	Scopes              []string       `json:"scopes"`
	PKCEEnabled         *bool          `json:"pkceEnabled,omitempty"`
	SPInitiated         bool           `json:"spInitiated"`
	CreateUnknownUsers  bool           `json:"createUnknownUsers"`
	DefaultUserRole     types.UserRole `json:"defaultUserRole"`
	AllowedEmailDomains []string       `json:"allowedEmailDomains"`
	CallbackURL         string         `json:"callbackUrl"`
}

type CustomOIDCConfigurationsResponse struct {
	Configurations []CustomOIDCConfiguration `json:"configurations"`
	// MembersWithOtherOrganizations are the members that a custom provider cannot authenticate, because
	// account exclusivity refuses an account that belongs to another organization as well.
	MembersWithOtherOrganizations []types.OrganizationMember `json:"membersWithOtherOrganizations"`
}

type CustomOIDCConfigurationRequest struct {
	CustomDomainID uuid.UUID `json:"customDomainId"`
	// CustomerOrganizationID targets a customer's own provider instead of the caller's organization.
	// Only a vendor or partner admin may set it; a customer caller may only ever target itself.
	CustomerOrganizationID *uuid.UUID     `json:"customerOrganizationId,omitempty"`
	Name                   string         `json:"name"`
	Slug                   string         `json:"slug"`
	Enabled                bool           `json:"enabled"`
	Issuer                 string         `json:"issuer"`
	ClientID               string         `json:"clientId"`
	ClientSecret           *string        `json:"clientSecret,omitempty"`
	Scopes                 []string       `json:"scopes"`
	PKCEEnabled            *bool          `json:"pkceEnabled,omitempty"`
	SPInitiated            bool           `json:"spInitiated"`
	CreateUnknownUsers     bool           `json:"createUnknownUsers"`
	DefaultUserRole        types.UserRole `json:"defaultUserRole"`
	AllowedEmailDomains    []string       `json:"allowedEmailDomains"`
}

func (r *CustomOIDCConfigurationRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Slug = validation.NormalizeSlug(r.Slug)
	r.Issuer = strings.TrimSpace(r.Issuer)
	r.ClientID = strings.TrimSpace(r.ClientID)

	r.Scopes = oidc.NormalizeScopes(r.Scopes)

	domains := make([]string, 0, len(r.AllowedEmailDomains))
	for _, domain := range r.AllowedEmailDomains {
		domain = validation.NormalizeHostname(strings.TrimPrefix(strings.TrimSpace(domain), "@"))
		if domain != "" && !slices.Contains(domains, domain) {
			domains = append(domains, domain)
		}
	}
	r.AllowedEmailDomains = domains
}

func (r *CustomOIDCConfigurationRequest) Validate() error {
	if r.CustomDomainID == uuid.Nil {
		return validation.NewValidationFailedError("customDomainId is required")
	}
	if r.Name == "" {
		return validation.NewValidationFailedError("name is required")
	}
	if len(r.Name) > customOIDCConfigurationNameMaxLength {
		return validation.NewValidationFailedError(
			fmt.Sprintf("name must be at most %v characters", customOIDCConfigurationNameMaxLength))
	}
	if r.Slug == "" {
		return validation.NewValidationFailedError("slug is required")
	}
	if err := validation.ValidateSlug(r.Slug); err != nil {
		return err
	}
	if _, err := oidc.ParseIssuerURL(r.Issuer); err != nil {
		return validation.NewValidationFailedError(err.Error())
	}
	if r.ClientID == "" {
		return validation.NewValidationFailedError("clientId is required")
	}
	if r.ClientSecret != nil && strings.TrimSpace(*r.ClientSecret) == "" {
		return validation.NewValidationFailedError(
			"clientSecret must not be empty. Omit it to keep the stored secret")
	}
	if _, err := types.ParseUserRole(string(r.DefaultUserRole)); err != nil {
		return validation.NewValidationFailedError("defaultUserRole is invalid")
	}
	for _, domain := range r.AllowedEmailDomains {
		if err := validation.ValidateHostname(domain); err != nil {
			return validation.NewValidationFailedError(
				fmt.Sprintf("allowedEmailDomains contains the invalid domain %q", domain))
		}
	}
	if r.CreateUnknownUsers && len(r.AllowedEmailDomains) == 0 {
		return validation.NewValidationFailedError(
			"allowedEmailDomains is required when accounts are created on first sign-in")
	}
	return nil
}
