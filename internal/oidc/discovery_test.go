package oidc_test

import (
	"testing"

	"github.com/distr-sh/distr/internal/oidc"
	. "github.com/onsi/gomega"
)

func TestParseIssuerURL(t *testing.T) {
	g := NewWithT(t)

	validIssuers := []string{
		"https://accounts.google.com",
		"https://login.microsoftonline.com/8eaef023-2b34-4da1-9baa-8bc8c9d6a490/v2.0",
		"https://acme.eu.auth0.com/",
		"https://keycloak.example.com/realms/acme",
		"https://acme.okta.com/oauth2/default",
	}
	for _, issuer := range validIssuers {
		_, err := oidc.ParseIssuerURL(issuer)
		g.Expect(err).ToNot(HaveOccurred(), issuer)
	}

	invalidIssuers := []string{
		"",
		"accounts.google.com",
		"http://accounts.google.com",
		"https://",
		"https://user:secret@accounts.google.com",
		"https://accounts.google.com?p=b2c_1_signin",
		"https://accounts.google.com#fragment",
		// The Entra ID endpoints without a tenant state a templated issuer that can never be verified.
		"https://login.microsoftonline.com/common/v2.0",
		"https://login.microsoftonline.com/organizations/v2.0",
		"https://login.microsoftonline.com/consumers/v2.0",
		"https://LOGIN.microsoftonline.com/Common/v2.0",
	}
	for _, issuer := range invalidIssuers {
		_, err := oidc.ParseIssuerURL(issuer)
		g.Expect(err).To(HaveOccurred(), issuer)
	}
}
