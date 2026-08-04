package api_test

import (
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func validCustomOIDCConfigurationRequest() api.CustomOIDCConfigurationRequest {
	return api.CustomOIDCConfigurationRequest{
		CustomDomainID:  uuid.New(),
		Name:            "Acme SSO",
		Issuer:          "https://acme.eu.auth0.com/",
		ClientID:        "client-id",
		ClientSecret:    new("secret"),
		DefaultUserRole: types.UserRoleReadWrite,
	}
}

func TestCustomOIDCConfigurationRequestNormalize(t *testing.T) {
	g := NewWithT(t)

	request := api.CustomOIDCConfigurationRequest{
		Name:                "  Acme SSO  ",
		Issuer:              "  https://acme.eu.auth0.com/  ",
		ClientID:            " client-id ",
		Scopes:              []string{" profile ", "email", "", "email", "openid"},
		AllowedEmailDomains: []string{" @Acme.com ", "acme.com", "https://sub.acme.com/path", ""},
	}
	request.Normalize()

	g.Expect(request.Name).To(Equal("Acme SSO"))
	g.Expect(request.Issuer).To(Equal("https://acme.eu.auth0.com/"))
	g.Expect(request.ClientID).To(Equal("client-id"))
	g.Expect(request.Scopes).To(Equal([]string{"openid", "profile", "email"}))
	g.Expect(request.AllowedEmailDomains).To(Equal([]string{"acme.com", "sub.acme.com"}))
}

func TestCustomOIDCConfigurationRequestNormalizeAddsOpenIDScope(t *testing.T) {
	g := NewWithT(t)

	request := api.CustomOIDCConfigurationRequest{}
	request.Normalize()

	g.Expect(request.Scopes).To(Equal([]string{"openid"}))
}

func TestCustomOIDCConfigurationRequestValidate(t *testing.T) {
	g := NewWithT(t)
	valid := validCustomOIDCConfigurationRequest()
	g.Expect(valid.Validate()).To(Succeed())

	invalid := map[string]func(r *api.CustomOIDCConfigurationRequest){
		"missing custom domain": func(r *api.CustomOIDCConfigurationRequest) { r.CustomDomainID = uuid.Nil },
		"missing name":          func(r *api.CustomOIDCConfigurationRequest) { r.Name = "" },
		"long name":             func(r *api.CustomOIDCConfigurationRequest) { r.Name = string(make([]byte, 101)) },
		"missing issuer":        func(r *api.CustomOIDCConfigurationRequest) { r.Issuer = "" },
		"insecure issuer":       func(r *api.CustomOIDCConfigurationRequest) { r.Issuer = "http://acme.example.com" },
		"missing client id":     func(r *api.CustomOIDCConfigurationRequest) { r.ClientID = "" },
		"blank client secret":   func(r *api.CustomOIDCConfigurationRequest) { r.ClientSecret = new("  ") },
		"unknown role":          func(r *api.CustomOIDCConfigurationRequest) { r.DefaultUserRole = "root" },
		"invalid email domain": func(r *api.CustomOIDCConfigurationRequest) {
			r.AllowedEmailDomains = []string{"not a domain"}
		},
		"provisioning without email domains": func(r *api.CustomOIDCConfigurationRequest) {
			r.CreateUnknownUsers = true
			r.AllowedEmailDomains = nil
		},
	}
	for name, modify := range invalid {
		request := validCustomOIDCConfigurationRequest()
		modify(&request)
		g.Expect(request.Validate()).To(HaveOccurred(), name)
	}

	withoutSecret := validCustomOIDCConfigurationRequest()
	withoutSecret.ClientSecret = nil
	g.Expect(withoutSecret.Validate()).To(Succeed())

	provisioning := validCustomOIDCConfigurationRequest()
	provisioning.CreateUnknownUsers = true
	provisioning.AllowedEmailDomains = []string{"acme.com"}
	g.Expect(provisioning.Validate()).To(Succeed())
}
