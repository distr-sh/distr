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

func TestAdvisoryStatusIsInitial(t *testing.T) {
	g := NewWithT(t)

	g.Expect(AdvisoryStatusTriage.IsInitial()).To(BeTrue())
	g.Expect(AdvisoryStatusDraft.IsInitial()).To(BeTrue())
	g.Expect(AdvisoryStatusPublished.IsInitial()).To(BeFalse())
	g.Expect(AdvisoryStatusResolved.IsInitial()).To(BeFalse())
	g.Expect(AdvisoryStatusCanceled.IsInitial()).To(BeFalse())
}

// Creating an advisory must never make it customer visible in one step.
func TestAdvisoryInitialStatusesAreNeverCustomerVisible(t *testing.T) {
	g := NewWithT(t)

	for _, status := range allAdvisoryStatuses {
		if status.IsInitial() {
			g.Expect(status.IsCustomerVisible()).To(BeFalse(), "status %q", status)
		}
	}
}

func TestAdvisoryStatusCanTransitionTo(t *testing.T) {
	allowed := map[AdvisoryStatus][]AdvisoryStatus{
		AdvisoryStatusTriage: {AdvisoryStatusDraft, AdvisoryStatusCanceled},
		AdvisoryStatusDraft: {
			AdvisoryStatusTriage, AdvisoryStatusPublished, AdvisoryStatusCanceled,
		},
		AdvisoryStatusPublished: {AdvisoryStatusDraft, AdvisoryStatusResolved},
		AdvisoryStatusResolved:  {AdvisoryStatusPublished},
		AdvisoryStatusCanceled:  {AdvisoryStatusDraft},
	}

	for _, from := range allAdvisoryStatuses {
		for _, to := range allAdvisoryStatuses {
			t.Run(string(from)+" to "+string(to), func(t *testing.T) {
				g := NewWithT(t)
				expected := false
				for _, target := range allowed[from] {
					if target == to {
						expected = true
					}
				}
				g.Expect(from.CanTransitionTo(to)).To(Equal(expected))
			})
		}
	}
}

// An advisory must pass through draft before it can become customer visible, and it must
// never become visible without an explicit publish step.
func TestAdvisoryStatusTriageCannotReachCustomerVisibleDirectly(t *testing.T) {
	g := NewWithT(t)

	g.Expect(AdvisoryStatusTriage.CanTransitionTo(AdvisoryStatusPublished)).To(BeFalse())
	g.Expect(AdvisoryStatusTriage.CanTransitionTo(AdvisoryStatusResolved)).To(BeFalse())
}

// Canceling is a decision not to disclose, so it must be unreachable once customers have
// already seen the advisory.
func TestAdvisoryStatusCannotCancelAfterDisclosure(t *testing.T) {
	g := NewWithT(t)

	g.Expect(AdvisoryStatusPublished.CanTransitionTo(AdvisoryStatusCanceled)).To(BeFalse())
	g.Expect(AdvisoryStatusResolved.CanTransitionTo(AdvisoryStatusCanceled)).To(BeFalse())
}

// Reopening a canceled advisory must not disclose it in one step, nor push it back into the
// triage inbox that is reserved for externally reported issues.
func TestAdvisoryStatusCanceledReopensIntoDraftOnly(t *testing.T) {
	g := NewWithT(t)

	g.Expect(AdvisoryStatusCanceled.CanTransitionTo(AdvisoryStatusDraft)).To(BeTrue())
	g.Expect(AdvisoryStatusCanceled.CanTransitionTo(AdvisoryStatusTriage)).To(BeFalse())
	g.Expect(AdvisoryStatusCanceled.CanTransitionTo(AdvisoryStatusPublished)).To(BeFalse())
	g.Expect(AdvisoryStatusCanceled.CanTransitionTo(AdvisoryStatusResolved)).To(BeFalse())
}

// No status may transition to itself: a no-op change would record a misleading timeline event.
func TestAdvisoryStatusCannotTransitionToItself(t *testing.T) {
	g := NewWithT(t)

	for _, status := range allAdvisoryStatuses {
		g.Expect(status.CanTransitionTo(status)).To(BeFalse(), "status %q", status)
	}
}

func TestAdvisoryStatusCanTransitionFromUnknownStatus(t *testing.T) {
	g := NewWithT(t)

	// A status read from a future schema version must not be treated as a valid source.
	unknown := AdvisoryStatus("withdrawn")
	g.Expect(unknown.CanTransitionTo(AdvisoryStatusDraft)).To(BeFalse())
}
