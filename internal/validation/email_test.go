package validation_test

import (
	"testing"

	"github.com/distr-sh/distr/internal/validation"
	. "github.com/onsi/gomega"
)

func TestValidateEmail(t *testing.T) {
	g := NewWithT(t)
	for _, email := range []string{
		"user@example.com",
		"first.last@mail.example.co.uk",
		"user+tag@example.com",
	} {
		g.Expect(validation.ValidateEmail(email)).To(Succeed(), email)
	}

	for _, email := range []string{
		"",
		"user@localhost",
		"user@example.com@allowed.io",
		"us er@example.com",
		".user@example.com",
		"user.@example.com",
		"first..last@example.com",
		"user@.example.com",
		"user@example.com.",
		"user@example..com",
	} {
		g.Expect(validation.ValidateEmail(email)).To(HaveOccurred(), email)
	}
}
