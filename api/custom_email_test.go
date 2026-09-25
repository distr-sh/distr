package api_test

import (
	"testing"

	"github.com/distr-sh/distr/api"
	. "github.com/onsi/gomega"
)

func TestCustomEmailSettingsNormalize(t *testing.T) {
	g := NewWithT(t)
	settings := api.CustomEmailSettings{SMTPHost: "smtps://SMTP.Example.Com:465/"}
	settings.Normalize()

	g.Expect(settings.SMTPHost).To(Equal("smtp.example.com"))
}
