package handlerutil_test

import (
	"strings"
	"testing"

	"github.com/distr-sh/distr/internal/handlerutil"
	. "github.com/onsi/gomega"
)

func TestValidateFilterRegex(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantErr bool
	}{
		{name: "empty filter", filter: "", wantErr: false},
		{name: "literal string", filter: "error", wantErr: false},
		{name: "simple wildcard", filter: "error.*", wantErr: false},
		{name: "alternation", filter: "error|warning", wantErr: false},
		{name: "anchored", filter: "^error$", wantErr: false},
		{name: "character class", filter: "[Ee]rror", wantErr: false},
		{name: "quantified group no inner quantifier", filter: "(foo|bar)+", wantErr: false},
		{name: "case insensitive flag", filter: "(?i)error", wantErr: false},
		{name: "plus quantifier", filter: "a+", wantErr: false},
		{name: "star quantifier", filter: "a*", wantErr: false},
		{name: "quest quantifier", filter: "a?", wantErr: false},
		{name: "repeat quantifier", filter: "a{2,5}", wantErr: false},
		{name: "nested plus in star group", filter: "(a+)*", wantErr: true},
		{name: "nested star in plus group", filter: "(a*)+", wantErr: true},
		{name: "nested plus in plus group", filter: "(a+)+", wantErr: true},
		{name: "nested quest in plus group", filter: "(a?)+", wantErr: true},
		{name: "nested wildcard quantifier", filter: "(.+)+", wantErr: true},
		{name: "deeply nested quantifier", filter: "((a+)b)+", wantErr: true},
		{name: "invalid syntax", filter: "(unclosed", wantErr: true},
		{name: "exceeds max length", filter: strings.Repeat("a", 201), wantErr: true},
		{name: "exactly max length", filter: strings.Repeat("a", 200), wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			err := handlerutil.ValidateFilterRegex(tt.filter)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}
