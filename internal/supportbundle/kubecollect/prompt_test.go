package kubecollect

import (
	"io"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

var namespaceOptions = []NamespaceOption{
	{Name: "default"},
	{Name: "kube-system"},
	{Name: "shop", HelmReleases: []string{"shop", "redis"}},
}

func TestParseNamespaceSelection(t *testing.T) {
	g := NewWithT(t)

	g.Expect(parseNamespaceSelection("  ", namespaceOptions, "shop")).To(Equal([]string{"shop"}))
	g.Expect(parseNamespaceSelection("3, kube-system 3,shop", namespaceOptions, "default")).
		To(Equal([]string{"shop", "kube-system"}))

	_, err := parseNamespaceSelection("4", namespaceOptions, "default")
	g.Expect(err).To(MatchError(ContainSubstring("4 is not a number from the list")))
	_, err = parseNamespaceSelection("1 monitoring", namespaceOptions, "default")
	g.Expect(err).To(MatchError(ContainSubstring(`"monitoring" is not a namespace from the list`)))
}

func TestPrompter(t *testing.T) {
	t.Run("asks again after an invalid answer", func(t *testing.T) {
		g := NewWithT(t)
		var out strings.Builder
		p := NewPrompter(strings.NewReader("monitoring\n2\nmaybe\nN\n"), &out)

		g.Expect(p.SelectNamespaces(namespaceOptions, "default")).To(Equal([]string{"kube-system"}))
		g.Expect(p.Confirm("Collect the logs?")).To(BeFalse())
		g.Expect(out.String()).To(ContainSubstring("3) shop (shop, redis)"))
		g.Expect(strings.Count(out.String(), "Namespaces to collect")).To(Equal(2))
		g.Expect(strings.Count(out.String(), "Collect the logs?")).To(Equal(2))
	})

	t.Run("takes the default for an empty answer", func(t *testing.T) {
		g := NewWithT(t)
		p := NewPrompter(strings.NewReader("\n\n"), io.Discard)

		g.Expect(p.SelectNamespaces(namespaceOptions, "shop")).To(Equal([]string{"shop"}))
		g.Expect(p.Confirm("Collect the logs?")).To(BeTrue())
	})

	t.Run("accepts a last answer without a newline", func(t *testing.T) {
		g := NewWithT(t)
		p := NewPrompter(strings.NewReader("no"), io.Discard)

		g.Expect(p.Confirm("Collect the logs?")).To(BeFalse())
	})

	t.Run("fails when the input ends", func(t *testing.T) {
		g := NewWithT(t)
		p := NewPrompter(strings.NewReader("monitoring\n"), io.Discard)

		_, err := p.SelectNamespaces(namespaceOptions, "default")
		g.Expect(err).To(MatchError(ErrNoAnswer))
	})
}
