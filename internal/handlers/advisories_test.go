package handlers

import (
	"fmt"
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestDetailChangeParts(t *testing.T) {
	before := func() types.Advisory {
		return types.Advisory{
			Title:       "Log parser crash",
			Description: "The parser crashes on malformed input.",
			Severity:    types.AdvisorySeverityMedium,
			CveID:       new(string),
		}
	}
	request := func(v types.Advisory) api.CreateUpdateAdvisoryRequest {
		return api.CreateUpdateAdvisoryRequest{
			Title:       v.Title,
			Description: v.Description,
			Severity:    string(v.Severity),
			CveID:       v.CveID,
		}
	}

	t.Run("is empty when nothing changed", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(detailChangeParts(before(), request(before()))).To(BeEmpty())
	})

	t.Run("reports a title change with both values", func(t *testing.T) {
		g := NewWithT(t)
		after := request(before())
		after.Title = "Remote code execution in the log parser"
		g.Expect(detailChangeParts(before(), after)).To(ConsistOf(
			`changed the title from "Log parser crash" to "Remote code execution in the log parser"`))
	})

	// The description is free-form Markdown, so the message says that it changed without
	// trying to show how.
	t.Run("reports the description without quoting it", func(t *testing.T) {
		g := NewWithT(t)
		after := request(before())
		after.Description = "The parser crashes and can be made to execute code."
		g.Expect(detailChangeParts(before(), after)).To(ConsistOf("updated the description"))
	})

	t.Run("reports every change", func(t *testing.T) {
		g := NewWithT(t)
		after := request(before())
		after.Title = "New title"
		after.Severity = string(types.AdvisorySeverityHigh)
		after.Description = "Rewritten."
		g.Expect(detailChangeParts(before(), after)).To(Equal([]string{
			`changed the title from "Log parser crash" to "New title"`,
			"changed the severity from medium to high",
			"updated the description",
		}))
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

func TestChangeParts(t *testing.T) {
	g := NewWithT(t)

	g.Expect(changeParts("tags", []string{"auth"}, []string{"auth"})).To(BeEmpty())
	g.Expect(changeParts("tags", []string{"auth"}, []string{"auth", "tls"})).
		To(ConsistOf("added the tags tls"))
	g.Expect(changeParts("tags", []string{"auth", "tls"}, nil)).
		To(ConsistOf("removed the tags auth, tls"))
	g.Expect(changeParts("references", []string{"https://a"}, []string{"https://b"})).
		To(Equal([]string{"added the references https://b", "removed the references https://a"}))
}

func TestVersionChangeParts(t *testing.T) {
	one := uuid.New()
	two := uuid.New()

	affected := func(id uuid.UUID, label string) versionMarking {
		return versionMarking{id: id, label: label, relation: types.AdvisoryVersionRelationAffected}
	}
	fixed := func(id uuid.UUID, label string) versionMarking {
		return versionMarking{id: id, label: label, relation: types.AdvisoryVersionRelationFixed}
	}

	t.Run("is empty when the markings are the same", func(t *testing.T) {
		g := NewWithT(t)
		markings := []versionMarking{affected(one, "MyApp 1.0.0")}
		g.Expect(versionChangeParts(markings, markings)).To(BeEmpty())
	})

	t.Run("reports a newly marked version", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(versionChangeParts(nil, []versionMarking{affected(one, "MyApp 1.0.0")})).
			To(ConsistOf("marked MyApp 1.0.0 as affected"))
	})

	t.Run("reports an unmarked version", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(versionChangeParts([]versionMarking{fixed(one, "MyApp 1.1.0")}, nil)).
			To(ConsistOf("unmarked MyApp 1.1.0"))
	})

	// Matching on id keeps this as one change rather than an unmark plus a mark, which is how
	// a reader thinks of a version that turned out to contain the fix.
	t.Run("reports a version that switched relation", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(versionChangeParts(
			[]versionMarking{affected(one, "MyApp 1.0.0")},
			[]versionMarking{fixed(one, "MyApp 1.0.0")},
		)).To(ConsistOf("changed MyApp 1.0.0 from affected to fixed"))
	})

	t.Run("reports additions and removals", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(versionChangeParts(
			[]versionMarking{affected(one, "MyApp 1.0.0")},
			[]versionMarking{fixed(two, "MyApp 1.1.0")},
		)).To(Equal([]string{"marked MyApp 1.1.0 as fixed", "unmarked MyApp 1.0.0"}))
	})

	t.Run("truncates a very long list", func(t *testing.T) {
		g := NewWithT(t)
		after := make([]versionMarking, 0, maxVersionChangeMessageParts+3)
		for i := range maxVersionChangeMessageParts + 3 {
			after = append(after, affected(uuid.New(), fmt.Sprintf("MyApp 1.0.%v", i)))
		}
		parts := versionChangeParts(nil, after)
		g.Expect(parts).To(HaveLen(maxVersionChangeMessageParts + 1))
		g.Expect(parts[len(parts)-1]).To(Equal("and 3 more"))
		g.Expect(parts).To(ContainElement("marked MyApp 1.0.0 as affected"))
		g.Expect(parts).NotTo(ContainElement(ContainSubstring("MyApp 1.0.10")))
	})
}
