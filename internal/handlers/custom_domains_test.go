package handlers

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestHostnameOf(t *testing.T) {
	g := NewWithT(t)
	for input, expected := range map[string]string{
		"app.distr.sh":                "app.distr.sh",
		"https://app.distr.sh":        "app.distr.sh",
		"https://app.distr.sh:8080":   "app.distr.sh",
		"http://localhost:8080":       "localhost",
		"registry.distr.sh:5000":      "registry.distr.sh",
		"":                            "",
		"https://app.distr.sh/portal": "app.distr.sh",
	} {
		g.Expect(hostnameOf(input)).To(Equal(expected), input)
	}
}

func TestNormalizeLegacyDomain(t *testing.T) {
	g := NewWithT(t)
	for input, expected := range map[string]string{
		"app.customer.com":          "app.customer.com",
		"https://app.customer.com":  "app.customer.com",
		"https://app.customer.com/": "app.customer.com",
		"  App.Customer.Com.  ":     "app.customer.com",
		"app.customer.com:443":      "app.customer.com",
	} {
		g.Expect(normalizeLegacyDomain(input)).To(Equal(expected), input)
	}
}

func TestIsPlatformOwnedDomain(t *testing.T) {
	g := NewWithT(t)
	g.Expect(isPlatformOwnedDomain("distr.sh")).To(BeTrue())
	g.Expect(isPlatformOwnedDomain("app.distr.sh")).To(BeTrue())
	g.Expect(isPlatformOwnedDomain("anything.deeply.nested.distr.sh")).To(BeTrue())
	g.Expect(isPlatformOwnedDomain("app.customer.com")).To(BeFalse())
	g.Expect(isPlatformOwnedDomain("notdistr.sh")).To(BeFalse())
}
