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
		g.Expect(legacy).To(ContainSubstring(`DISTR_CONTROLLER_NAME: "agent"`))
		g.Expect(legacy).NotTo(ContainSubstring("distr-controller"))

		current := renderKubernetesManifest(g, false, scope)
		g.Expect(current).To(ContainSubstring("name: distr-controller\n"))
		g.Expect(current).To(ContainSubstring(`DISTR_CONTROLLER_NAME: "controller"`))
		g.Expect(current).NotTo(ContainSubstring("distr-agent"))
	}
}

func renderRegistrySettings(g *WithT, deploymentType types.DeploymentType, distrHost, registryHost string) string {
	dt := types.DeploymentTargetFull{
		DeploymentTarget: types.DeploymentTarget{Type: deploymentType},
		ControllerVersion: types.ControllerVersion{
			ComposeFileRevision:  types.CurrentComposeFileRevision,
			ManifestFileRevision: types.CurrentManifestFileRevision,
		},
	}
	tmpl, err := getTemplate(dt)
	g.Expect(err).NotTo(HaveOccurred())
	var buf bytes.Buffer
	g.Expect(tmpl.Execute(&buf, map[string]any{
		"controllerName":         controllerName(dt),
		"controllerDockerConfig": "e30=",
		"targetSecret":           "secret",
		"targetNamespace":        "default",
		"targetScope":            types.DeploymentTargetScopeNamespace,
		"registryEnabled":        true,
		"registryHost":           registryHost,
		"registryPlainHttp":      registryPlainHTTP(distrHost),
	})).To(Succeed())
	return buf.String()
}

func TestControllerManifestsPullOverPlainHTTPOnlyFromAnHTTPDistr(t *testing.T) {
	g := NewWithT(t)

	docker := renderRegistrySettings(g, types.DeploymentTypeDocker, "http://localhost:8080", "localhost:8585")
	g.Expect(docker).To(ContainSubstring("DISTR_REGISTRY_HOST: 'localhost:8585'"))
	g.Expect(docker).To(ContainSubstring("DISTR_REGISTRY_PLAIN_HTTP: 'true'"))
	kubernetes := renderRegistrySettings(g, types.DeploymentTypeKubernetes, "http://localhost:8080", "localhost:8585")
	g.Expect(kubernetes).To(ContainSubstring(`DISTR_REGISTRY_PLAIN_HTTP: "true"`))

	docker = renderRegistrySettings(g, types.DeploymentTypeDocker, "https://app.distr.sh", "registry.distr.sh")
	g.Expect(docker).To(ContainSubstring("DISTR_REGISTRY_PLAIN_HTTP: 'false'"))
	kubernetes = renderRegistrySettings(g, types.DeploymentTypeKubernetes, "https://app.distr.sh", "registry.distr.sh")
	g.Expect(kubernetes).To(ContainSubstring(`DISTR_REGISTRY_PLAIN_HTTP: "false"`))
}
