package controllermanifest

import (
	"bytes"
	"testing"

	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func render(g *WithT, dt types.DeploymentTargetFull, data map[string]any) string {
	tmpl, err := getTemplate(dt)
	g.Expect(err).NotTo(HaveOccurred())
	var buf bytes.Buffer
	g.Expect(tmpl.Execute(&buf, data)).To(Succeed())
	return buf.String()
}

func currentTarget(deploymentType types.DeploymentType, legacy bool) types.DeploymentTargetFull {
	return types.DeploymentTargetFull{
		DeploymentTarget: types.DeploymentTarget{Type: deploymentType, LegacyControllerName: legacy},
		ControllerVersion: types.ControllerVersion{
			Name:                 "1.2.3",
			ManifestFileRevision: types.CurrentManifestFileRevision,
			ComposeFileRevision:  types.CurrentComposeFileRevision,
		},
	}
}

func templateData(dt types.DeploymentTargetFull) map[string]any {
	return map[string]any{
		"controllerName":            controllerName(dt),
		"controllerDockerConfig":    "e30=",
		"controllerImageRepository": "registry.example.com/mirror",
		"controllerVersion":         dt.ControllerVersion.Name,
		"targetSecret":              "secret",
		"targetNamespace":           "default",
	}
}

func renderKubernetesManifest(g *WithT, legacy bool, scope types.DeploymentTargetScope) string {
	dt := currentTarget(types.DeploymentTypeKubernetes, legacy)
	data := templateData(dt)
	data["targetScope"] = scope
	return render(g, dt, data)
}

func TestCurrentKubernetesManifestKeepsAgentNamesOfLegacyTargets(t *testing.T) {
	g := NewWithT(t)
	for _, scope := range []types.DeploymentTargetScope{
		types.DeploymentTargetScopeCluster, types.DeploymentTargetScopeNamespace,
	} {
		legacy := renderKubernetesManifest(g, true, scope)
		g.Expect(legacy).To(ContainSubstring("name: distr-agent\n"))
		g.Expect(legacy).To(ContainSubstring(`DISTR_CONTROLLER_NAME: "agent"`))
		g.Expect(legacy).NotTo(ContainSubstring("distr-controller"))

		current := renderKubernetesManifest(g, false, scope)
		g.Expect(current).To(ContainSubstring("name: distr-controller\n"))
		g.Expect(current).To(ContainSubstring(`DISTR_CONTROLLER_NAME: "controller"`))
		g.Expect(current).NotTo(ContainSubstring("distr-agent"))
	}
}

func TestCurrentManifestsPullControllerImagesFromRepository(t *testing.T) {
	g := NewWithT(t)

	kubernetes := renderKubernetesManifest(g, false, types.DeploymentTargetScopeCluster)
	g.Expect(kubernetes).To(ContainSubstring(`image: "registry.example.com/mirror/kubernetes-controller:1.2.3"`))

	dt := currentTarget(types.DeploymentTypeDocker, false)
	docker := render(g, dt, templateData(dt))
	g.Expect(docker).To(ContainSubstring(`image: 'registry.example.com/mirror/docker-controller:1.2.3'`))
	g.Expect(docker).To(ContainSubstring(`image: 'registry.example.com/mirror/docker-autoheal:1.2.3'`))
}

func TestImageRepositoryOnDomainKeepsPathOfConfiguredRepository(t *testing.T) {
	g := NewWithT(t)
	g.Expect(imageRepositoryOnDomain("registry.distr.sh:5000/distr-sh/distr", "registry.vendor.com")).
		To(Equal("registry.vendor.com/distr-sh/distr"))
	g.Expect(imageRepositoryOnDomain("localhost:5000", "registry.vendor.com")).To(Equal("registry.vendor.com"))
}
