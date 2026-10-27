package validation_test

import (
	"strings"
	"testing"

	"github.com/distr-sh/distr/internal/validation"
	. "github.com/onsi/gomega"
)

func TestParseRegex(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{name: "RE2 inline flag", pattern: `(?i)^release-`},
		{name: "nested quantifier", pattern: `^(\d+\.)+\d+$`},
		{name: "invalid syntax", pattern: `(unclosed`, wantErr: true},
		{name: "maximum length", pattern: strings.Repeat("a", 200)},
		{name: "exceeds maximum length", pattern: strings.Repeat("a", 201), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			_, err := validation.ParseRegex(tt.pattern, 200)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}
