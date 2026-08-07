package oidc_test

import (
	"testing"

	"github.com/distr-sh/distr/internal/oidc"
	. "github.com/onsi/gomega"
)

func TestNormalizeScopes(t *testing.T) {
	g := NewWithT(t)

	expected := map[string][]string{
		"":                   {"openid"},
		"email":              {"openid", "email"},
		"email,profile":      {"openid", "email", "profile"},
		"email profile":      {"openid", "email", "profile"},
		"openid, email":      {"openid", "email"},
		" email , profile  ": {"openid", "email", "profile"},
	}
	for value, scopes := range expected {
		g.Expect(oidc.NormalizeScopes([]string{value})).To(Equal(scopes), value)
	}

	g.Expect(oidc.NormalizeScopes([]string{"profile", "email", "email", ""})).
		To(Equal([]string{"openid", "profile", "email"}))
	g.Expect(oidc.NormalizeScopes(nil)).To(Equal([]string{"openid"}))
}
