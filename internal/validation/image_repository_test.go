package validation_test

import (
	"testing"

	"github.com/distr-sh/distr/internal/validation"
	. "github.com/onsi/gomega"
)

func TestValidateImageRepository(t *testing.T) {
	g := NewWithT(t)
	for _, value := range []string{
		"ghcr.io/distr-sh/distr",
		"harbor.example.com",
		"harbor.example.com:5000/mirror/distr-sh/distr",
		"registry:5000/distr",
		"localhost:5000/distr",
		"localhost/distr",
	} {
		g.Expect(validation.ValidateImageRepository(value)).To(Succeed(), value)
	}

	for _, value := range []string{
		"",
		"distr-sh/distr",
		"registry/distr",
		"ghcr.io/distr-sh/distr:1.0.0",
		"ghcr.io/distr-sh/distr@sha256:abc",
		"ghcr.io/Distr-sh/distr",
		"ghcr.io//distr",
		"ghcr.io/distr sh",
	} {
		g.Expect(validation.ValidateImageRepository(value)).To(HaveOccurred(), value)
	}
}

func TestNormalizeImageRepository(t *testing.T) {
	g := NewWithT(t)
	for input, expected := range map[string]string{
		"ghcr.io/distr-sh/distr":            "ghcr.io/distr-sh/distr",
		"  https://harbor.example.com/x/  ": "harbor.example.com/x",
		"http://localhost:5000":             "localhost:5000",
	} {
		g.Expect(validation.NormalizeImageRepository(input)).To(Equal(expected), input)
	}
}
