package controllermanifest

import (
	"bytes"
	"testing"

	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func renderKubernetesManifest(g *WithT, legacy bool, scope types.DeploymentTargetScope) string {
	dt := types.DeploymentTargetFull{
		DeploymentTarget:  types.DeploymentTarget{Type: types.DeploymentTypeKubernetes, LegacyControllerName: legacy},
		ControllerVersion: types.ControllerVersion{ManifestFileRevision: types.CurrentManifestFileRevision},
	}
	tmpl, err := getTemplate(dt)
	g.Expect(err).NotTo(HaveOccurred())
	var buf bytes.Buffer
	g.Expect(tmpl.Execute(&buf, map[string]any{
		"controllerName":         controllerName(dt),
		"controllerDockerConfig": "e30=",
		"targetSecret":           "secret",
		"targetNamespace":        "default",
		"targetScope":            scope,
	})).To(Succeed())
	return buf.String()
}

func TestCurrentKubernetesManifestKeepsAgentNamesOfLegacyTargets(t *testing.T) {
	g := NewWithT(t)
	for _, scope := range []types.DeploymentTargetScope{
		types.DeploymentTargetScopeCluster, types.DeploymentTargetScopeNamespace,
	} {
		legacy := renderKubernetesManifest(g, true, scope)
		g.Expect(legacy).To(ContainSubstring("name: distr-agent\n"))
		g.Expect(legacy).NotTo(ContainSubstring("distr-controller"))

		current := renderKubernetesManifest(g, false, scope)
		g.Expect(current).To(ContainSubstring("name: distr-controller\n"))
		g.Expect(current).NotTo(ContainSubstring("distr-agent"))
	}
}
