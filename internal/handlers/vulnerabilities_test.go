package handlers

import (
	"fmt"
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestDetailChangeMessage(t *testing.T) {
	before := func() types.Vulnerability {
		return types.Vulnerability{
			Title:       "Log parser crash",
			Description: "The parser crashes on malformed input.",
			Severity:    types.VulnerabilitySeverityMedium,
			CveID:       new(string),
		}
	}
	request := func(v types.Vulnerability) api.CreateUpdateVulnerabilityRequest {
		return api.CreateUpdateVulnerabilityRequest{
			Title:       v.Title,
			Description: v.Description,
			Severity:    string(v.Severity),
			CveID:       v.CveID,
		}
	}

	t.Run("is nil when nothing changed", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(detailChangeMessage(before(), request(before()))).To(BeNil())
	})

	t.Run("reports a title change with both values", func(t *testing.T) {
		g := NewWithT(t)
		after := request(before())
		after.Title = "Remote code execution in the log parser"
		g.Expect(detailChangeMessage(before(), after)).To(HaveValue(Equal(
			`changed the title from "Log parser crash" to "Remote code execution in the log parser"`)))
	})

	t.Run("reports a severity change", func(t *testing.T) {
		g := NewWithT(t)
		after := request(before())
		after.Severity = string(types.VulnerabilitySeverityCritical)
		g.Expect(detailChangeMessage(before(), after)).
			To(HaveValue(Equal("changed the severity from medium to critical")))
	})

	// The description is free-form Markdown, so the message says that it changed without
	// trying to show how.
	t.Run("reports the description without quoting it", func(t *testing.T) {
		g := NewWithT(t)
		after := request(before())
		after.Description = "The parser crashes and can be made to execute code."
		g.Expect(detailChangeMessage(before(), after)).To(HaveValue(Equal("updated the description")))
	})

	t.Run("joins several changes", func(t *testing.T) {
		g := NewWithT(t)
		after := request(before())
		after.Title = "New title"
		after.Severity = string(types.VulnerabilitySeverityHigh)
		after.Description = "Rewritten."
		g.Expect(detailChangeMessage(before(), after)).To(HaveValue(Equal(
			`changed the title from "Log parser crash" to "New title"; ` +
				"changed the severity from medium to high; updated the description")))
	})
}

func TestCveChangeMessage(t *testing.T) {
	g := NewWithT(t)
	first := "CVE-2024-0001"
	second := "CVE-2024-0002"

	g.Expect(cveChangeMessage(nil, nil)).To(BeEmpty())
	g.Expect(cveChangeMessage(&first, &first)).To(BeEmpty())
	g.Expect(cveChangeMessage(nil, &first)).To(Equal("set the CVE ID to CVE-2024-0001"))
	g.Expect(cveChangeMessage(&first, nil)).To(Equal("removed the CVE ID CVE-2024-0001"))
	g.Expect(cveChangeMessage(&first, &second)).
		To(Equal("changed the CVE ID from CVE-2024-0001 to CVE-2024-0002"))
}

func TestVersionChangeMessage(t *testing.T) {
	one := uuid.New()
	two := uuid.New()

	affected := func(id uuid.UUID, label string) versionMarking {
		return versionMarking{id: id, label: label, relation: types.VulnerabilityVersionRelationAffected}
	}
	fixed := func(id uuid.UUID, label string) versionMarking {
		return versionMarking{id: id, label: label, relation: types.VulnerabilityVersionRelationFixed}
	}

	t.Run("is nil when the markings are the same", func(t *testing.T) {
		g := NewWithT(t)
		markings := []versionMarking{affected(one, "MyApp 1.0.0")}
		g.Expect(versionChangeMessage(markings, markings)).To(BeNil())
	})

	t.Run("reports a newly marked version", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(versionChangeMessage(nil, []versionMarking{affected(one, "MyApp 1.0.0")})).
			To(HaveValue(Equal("marked MyApp 1.0.0 as affected")))
	})

	t.Run("reports an unmarked version", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(versionChangeMessage([]versionMarking{fixed(one, "MyApp 1.1.0")}, nil)).
			To(HaveValue(Equal("unmarked MyApp 1.1.0")))
	})

	// Matching on id keeps this as one change rather than an unmark plus a mark, which is how
	// a reader thinks of a version that turned out to contain the fix.
	t.Run("reports a version that switched relation", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(versionChangeMessage(
			[]versionMarking{affected(one, "MyApp 1.0.0")},
			[]versionMarking{fixed(one, "MyApp 1.0.0")},
		)).To(HaveValue(Equal("changed MyApp 1.0.0 from affected to fixed")))
	})

	t.Run("joins additions and removals", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(versionChangeMessage(
			[]versionMarking{affected(one, "MyApp 1.0.0")},
			[]versionMarking{fixed(two, "MyApp 1.1.0")},
		)).To(HaveValue(Equal("marked MyApp 1.1.0 as fixed; unmarked MyApp 1.0.0")))
	})

	t.Run("truncates a very long message", func(t *testing.T) {
		g := NewWithT(t)
		after := make([]versionMarking, 0, maxVersionChangeMessageParts+3)
		for i := range maxVersionChangeMessageParts + 3 {
			after = append(after, affected(uuid.New(), fmt.Sprintf("MyApp 1.0.%v", i)))
		}
		message := versionChangeMessage(nil, after)
		g.Expect(message).NotTo(BeNil())
		g.Expect(*message).To(HaveSuffix("; and 3 more"))
		g.Expect(*message).To(ContainSubstring("marked MyApp 1.0.0 as affected"))
		g.Expect(*message).NotTo(ContainSubstring("MyApp 1.0.10"))
	})
}
