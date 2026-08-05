package api_test

import (
	"testing"

	"github.com/distr-sh/distr/api"
	. "github.com/onsi/gomega"
)

func TestCustomEmailSettingsNormalize(t *testing.T) {
	g := NewWithT(t)
	settings := api.CustomEmailSettings{
		FromAddress:  "  noreply@example.com  ",
		SMTPHost:     " smtps://SMTP.Example.Com:465/ ",
		SMTPUsername: "  apikey  ",
	}
	settings.Normalize()

	g.Expect(settings.FromAddress).To(Equal("noreply@example.com"))
	g.Expect(settings.SMTPHost).To(Equal("smtp.example.com"))
	g.Expect(settings.SMTPUsername).To(Equal("apikey"))
}
