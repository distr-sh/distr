package security_test

import (
	"regexp"
	"testing"

	"github.com/distr-sh/distr/internal/security"
	. "github.com/onsi/gomega"
)

func TestGenerateRecoveryCodes(t *testing.T) {
	g := NewWithT(t)

	codes, err := security.GenerateRecoveryCodes()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(codes).To(HaveLen(10))

	for _, code := range codes {
		g.Expect(code).To(HaveLen(10))
		g.Expect(code).To(MatchRegexp("^[0-9a-f]{10}$"))
	}
}

func TestGenerateRecoveryCodes_Uniqueness(t *testing.T) {
	g := NewWithT(t)

	codes, err := security.GenerateRecoveryCodes()
	g.Expect(err).NotTo(HaveOccurred())

	seen := make(map[string]bool)
	for _, code := range codes {
		g.Expect(seen[code]).To(BeFalse(), "duplicate code found: %s", code)
		seen[code] = true
	}
}

func TestHashRecoveryCode(t *testing.T) {
	g := NewWithT(t)

	code := "1234567890"
	salt, hash, err := security.HashRecoveryCode(code)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(salt).NotTo(BeEmpty())
	g.Expect(hash).NotTo(BeEmpty())
}

func TestHashRecoveryCode_UniqueSalts(t *testing.T) {
	g := NewWithT(t)

	code := "1234567890"
	salt1, hash1, err := security.HashRecoveryCode(code)
	g.Expect(err).NotTo(HaveOccurred())

	salt2, hash2, err := security.HashRecoveryCode(code)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(salt1).NotTo(Equal(salt2))
	g.Expect(hash1).NotTo(Equal(hash2))
}

func TestHashRecoveryCode_Normalization(t *testing.T) {
	g := NewWithT(t)

	salt1, hash1, err := security.HashRecoveryCode("12345-67890")
	g.Expect(err).NotTo(HaveOccurred())

	salt2, hash2, err := security.HashRecoveryCode("1234567890")
	g.Expect(err).NotTo(HaveOccurred())

	verified := security.VerifyRecoveryCode("1234567890", salt1, hash1)
	g.Expect(verified).To(BeTrue())

	verified = security.VerifyRecoveryCode("12345-67890", salt2, hash2)
	g.Expect(verified).To(BeTrue())
}

func TestVerifyRecoveryCode(t *testing.T) {
	g := NewWithT(t)

	code := "1234567890"
	salt, hash, err := security.HashRecoveryCode(code)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(security.VerifyRecoveryCode(code, salt, hash)).To(BeTrue())
	g.Expect(security.VerifyRecoveryCode("0987654321", salt, hash)).To(BeFalse())
}

func TestVerifyRecoveryCode_WithFormatting(t *testing.T) {
	g := NewWithT(t)

	code := "1234567890"
	salt, hash, err := security.HashRecoveryCode(code)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(security.VerifyRecoveryCode("12345-67890", salt, hash)).To(BeTrue())
	g.Expect(security.VerifyRecoveryCode("1234567890", salt, hash)).To(BeTrue())
}

func TestVerifyRecoveryCode_CaseInsensitive(t *testing.T) {
	g := NewWithT(t)

	code := "abcdef1234"
	salt, hash, err := security.HashRecoveryCode(code)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(security.VerifyRecoveryCode("ABCDEF1234", salt, hash)).To(BeTrue())
	g.Expect(security.VerifyRecoveryCode("AbCdEf1234", salt, hash)).To(BeTrue())
	g.Expect(security.VerifyRecoveryCode("abcdef1234", salt, hash)).To(BeTrue())
}

func TestNormalizeRecoveryCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with hyphen",
			input:    "12345-67890",
			expected: "1234567890",
		},
		{
			name:     "without hyphen",
			input:    "1234567890",
			expected: "1234567890",
		},
		{
			name:     "uppercase",
			input:    "ABCDEF1234",
			expected: "abcdef1234",
		},
		{
			name:     "mixed case with hyphen",
			input:    "AbCdE-F1234",
			expected: "abcdef1234",
		},
		{
			name:     "multiple hyphens",
			input:    "12-34-56-78-90",
			expected: "1234567890",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := security.NormalizeRecoveryCode(tt.input)
			g.Expect(result).To(Equal(tt.expected))
		})
	}
}

func TestFormatRecoveryCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid 10-character code",
			input:    "1234567890",
			expected: "12345-67890",
		},
		{
			name:     "hex code",
			input:    "abcdef1234",
			expected: "abcde-f1234",
		},
		{
			name:     "too short",
			input:    "12345",
			expected: "12345",
		},
		{
			name:     "too long",
			input:    "12345678901",
			expected: "12345678901",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "already formatted",
			input:    "12345-67890",
			expected: "12345-67890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := security.FormatRecoveryCode(tt.input)
			g.Expect(result).To(Equal(tt.expected))
		})
	}
}

func TestRecoveryCodeFormat_Integration(t *testing.T) {
	g := NewWithT(t)

	codes, err := security.GenerateRecoveryCodes()
	g.Expect(err).NotTo(HaveOccurred())

	hexPattern := regexp.MustCompile("^[0-9a-f]{10}$")
	formattedPattern := regexp.MustCompile("^[0-9a-f]{5}-[0-9a-f]{5}$")

	for _, code := range codes {
		g.Expect(hexPattern.MatchString(code)).To(BeTrue())

		formatted := security.FormatRecoveryCode(code)
		g.Expect(formattedPattern.MatchString(formatted)).To(BeTrue())

		normalized := security.NormalizeRecoveryCode(formatted)
		g.Expect(normalized).To(Equal(code))
	}
}
