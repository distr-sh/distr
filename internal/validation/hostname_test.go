package validation_test

import (
	"strings"
	"testing"

	"github.com/distr-sh/distr/internal/validation"
	. "github.com/onsi/gomega"
)

func TestValidateHostname(t *testing.T) {
	g := NewWithT(t)
	validHostnames := []string{
		"example.com",
		"app.example.com",
		"registry.some-company.co.uk",
		"a.b",
		"1.2.3.4.example.com",
		"xn--bcher-kva.example",
	}
	for _, hostname := range validHostnames {
		g.Expect(validation.ValidateHostname(hostname)).To(Succeed(), hostname)
	}

	invalidHostnames := []string{
		"",
		"example",
		"example.com.",
		".example.com",
		"-example.com",
		"example-.com",
		"exa mple.com",
		"Example.com",
		"https://example.com",
		"example.com/path",
		"example.com:8080",
		"foo_bar.example.com",
		strings.Repeat("a", 63) + "a.example.com",
		strings.Repeat("a.", 127) + "example.com",
	}
	for _, hostname := range invalidHostnames {
		g.Expect(validation.ValidateHostname(hostname)).To(HaveOccurred(), hostname)
	}
}

func TestNormalizeHostname(t *testing.T) {
	g := NewWithT(t)
	for input, expected := range map[string]string{
		"example.com":       "example.com",
		"Example.COM":       "example.com",
		"  example.com  ":   "example.com",
		"example.com.":      "example.com",
		"App.Example.Com. ": "app.example.com",
	} {
		g.Expect(validation.NormalizeHostname(input)).To(Equal(expected), input)
		g.Expect(validation.ValidateHostname(validation.NormalizeHostname(input))).To(Succeed(), input)
	}
}
