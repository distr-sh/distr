package validation_test

import (
	"testing"

	"github.com/distr-sh/distr/internal/validation"
	. "github.com/onsi/gomega"
)

type trimRole string

type trimNested struct {
	Label   string
	Aliases []string
	Labels  map[string]string
}

type trimRequest struct {
	Name        string
	Description *string
	Role        trimRole
	Payload     []byte
	Nested      trimNested
	Items       []trimNested
	Password    string
	SecretValue string `trim:"-"`
	unexported  string
}

func TestTrimStrings(t *testing.T) {
	g := NewWithT(t)
	description := "  a description\n"
	request := trimRequest{
		Name:        "  Acme Inc. ",
		Description: &description,
		Role:        "  vendor ",
		Payload:     []byte("  raw bytes  "),
		Nested:      trimNested{Label: " nested ", Aliases: []string{" one ", "two"}, Labels: map[string]string{"k": " v "}},
		Items:       []trimNested{{Label: " item "}},
		Password:    "  a password  ",
		SecretValue: "  a secret value\n",
		unexported:  " untouched ",
	}

	validation.TrimStrings(&request)

	g.Expect(request.Name).To(Equal("Acme Inc."))
	g.Expect(*request.Description).To(Equal("a description"))
	g.Expect(request.Role).To(Equal(trimRole("vendor")))
	g.Expect(request.Nested.Label).To(Equal("nested"))
	g.Expect(request.Nested.Aliases).To(Equal([]string{"one", "two"}))
	g.Expect(request.Nested.Labels).To(Equal(map[string]string{"k": "v"}))
	g.Expect(request.Items[0].Label).To(Equal("item"))
	g.Expect(request.Password).To(Equal("a password"))

	g.Expect(string(request.Payload)).To(Equal("  raw bytes  "))
	g.Expect(request.SecretValue).To(Equal("  a secret value\n"))
	g.Expect(request.unexported).To(Equal(" untouched "))
}
