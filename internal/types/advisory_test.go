package types

import (
	"testing"

	. "github.com/onsi/gomega"
)

var allAdvisoryStatuses = []AdvisoryStatus{
	AdvisoryStatusTriage,
	AdvisoryStatusDraft,
	AdvisoryStatusPublished,
	AdvisoryStatusResolved,
	AdvisoryStatusCanceled,
}

func TestParseAdvisoryStatus(t *testing.T) {
	g := NewWithT(t)

	for _, status := range allAdvisoryStatuses {
		parsed, err := ParseAdvisoryStatus(string(status))
		g.Expect(err).NotTo(HaveOccurred(), "status %q", status)
		g.Expect(parsed).To(Equal(status))
	}

	// Casing and whitespace are not normalized here: callers must send the exact value.
	for _, value := range []string{"", "Draft", " draft", "unknown", "PUBLISHED", "cancelled"} {
		_, err := ParseAdvisoryStatus(value)
		g.Expect(err).To(MatchError(ErrInvalidAdvisoryStatus), "value %q", value)
	}
}

func TestParseAdvisorySeverity(t *testing.T) {
	g := NewWithT(t)

	valid := []AdvisorySeverity{
		AdvisorySeverityNone,
		AdvisorySeverityLow,
		AdvisorySeverityMedium,
		AdvisorySeverityHigh,
		AdvisorySeverityCritical,
	}
	for _, severity := range valid {
		parsed, err := ParseAdvisorySeverity(string(severity))
		g.Expect(err).NotTo(HaveOccurred(), "severity %q", severity)
		g.Expect(parsed).To(Equal(severity))
	}

	for _, value := range []string{"", "Critical", "informational", "9.8"} {
		_, err := ParseAdvisorySeverity(value)
		g.Expect(err).To(MatchError(ErrInvalidAdvisorySeverity), "value %q", value)
	}
}

func TestAdvisoryStatusIsCustomerVisible(t *testing.T) {
	g := NewWithT(t)

	g.Expect(AdvisoryStatusTriage.IsCustomerVisible()).To(BeFalse())
	g.Expect(AdvisoryStatusDraft.IsCustomerVisible()).To(BeFalse())
	g.Expect(AdvisoryStatusPublished.IsCustomerVisible()).To(BeTrue())
	g.Expect(AdvisoryStatusResolved.IsCustomerVisible()).To(BeTrue())
	g.Expect(AdvisoryStatusCanceled.IsCustomerVisible()).To(BeFalse())
}
