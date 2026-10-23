package chartdigest

import (
	"testing"

	"github.com/distr-sh/distr/internal/registry/name"
	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func TestParseChartReference(t *testing.T) {
	g := NewWithT(t)
	version := func(chartType types.HelmChartType, url string) types.ApplicationVersion {
		return types.ApplicationVersion{ChartType: &chartType, ChartUrl: &url, ChartVersion: new("1.0.0")}
	}

	g.Expect(parseChartReference(version(types.HelmChartTypeOCI, "oci://Registry.Example.com:8585/acme/charts/app"))).
		To(Equal(&chartReference{
			host: "registry.example.com",
			name: name.Name{OrgName: "acme", ArtifactName: "charts/app"},
			tag:  "1.0.0",
		}))

	for _, url := range []string{
		"oci://registry.example.com/acme",
		"oci://registry.example.com/acme/app:1.0.0",
		"oci://registry.example.com/acme/app@sha256:aaaa",
		"https://registry.example.com/acme/app",
	} {
		g.Expect(parseChartReference(version(types.HelmChartTypeOCI, url))).To(BeNil(), url)
	}
	g.Expect(parseChartReference(version(types.HelmChartTypeRepository, "oci://registry.example.com/acme/app"))).
		To(BeNil())
}
