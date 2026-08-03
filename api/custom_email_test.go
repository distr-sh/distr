package api_test

import (
	"testing"

	"github.com/distr-sh/distr/api"
	. "github.com/onsi/gomega"
)

func validCustomEmailSettings() api.CustomEmailSettings {
	return api.CustomEmailSettings{
		FromAddress: "noreply@example.com",
		SMTPHost:    "smtp.example.com",
		SMTPPort:    587,
	}
}

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

func TestCustomEmailSettingsValidate(t *testing.T) {
	g := NewWithT(t)

	valid := validCustomEmailSettings()
	g.Expect(valid.Validate()).To(Succeed())

	withName := validCustomEmailSettings()
	withName.FromAddress = "Example Support <support@example.com>"
	g.Expect(withName.Validate()).To(Succeed())

	invalid := map[string]func(*api.CustomEmailSettings){
		"empty from address":   func(s *api.CustomEmailSettings) { s.FromAddress = "" },
		"from address is name": func(s *api.CustomEmailSettings) { s.FromAddress = "noreply" },
		"empty host":           func(s *api.CustomEmailSettings) { s.SMTPHost = "" },
		"host with scheme":     func(s *api.CustomEmailSettings) { s.SMTPHost = "https://smtp.example.com" },
		"host with port":       func(s *api.CustomEmailSettings) { s.SMTPHost = "smtp.example.com:587" },
		"port zero":            func(s *api.CustomEmailSettings) { s.SMTPPort = 0 },
		"port negative":        func(s *api.CustomEmailSettings) { s.SMTPPort = -1 },
		"port too large":       func(s *api.CustomEmailSettings) { s.SMTPPort = 65536 },
	}
	for name, modify := range invalid {
		settings := validCustomEmailSettings()
		modify(&settings)
		g.Expect(settings.Validate()).To(HaveOccurred(), name)
	}
}
