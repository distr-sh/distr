package controllerconnect

import (
	"testing"

	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func orgWithScripts(features ...types.Feature) types.OrganizationWithBranding {
	return types.OrganizationWithBranding{Organization: types.Organization{
		Features:                    features,
		DockerPreConnectScript:      new("check-docker"),
		DockerPostConnectScript:     new("after-docker"),
		KubernetesPreConnectScript:  new("check-kubernetes"),
		KubernetesPostConnectScript: new("after-kubernetes"),
	}}
}

func TestUsesConnectScript(t *testing.T) {
	g := NewWithT(t)
	docker := types.DeploymentTarget{Type: types.DeploymentTypeDocker}
	kubernetes := types.DeploymentTarget{Type: types.DeploymentTypeKubernetes, Namespace: new("default")}

	g.Expect(usesConnectScript(docker, orgWithScripts())).To(BeFalse())
	g.Expect(usesConnectScript(kubernetes, orgWithScripts())).To(BeFalse())

	enabled := orgWithScripts(types.FeaturePrePostScripts)
	g.Expect(usesConnectScript(docker, enabled)).To(BeTrue())
	g.Expect(usesConnectScript(kubernetes, enabled)).To(BeTrue())

	blank := types.OrganizationWithBranding{Organization: types.Organization{
		Features:                   []types.Feature{types.FeaturePrePostScripts},
		DockerPreConnectScript:     new("  \n"),
		KubernetesPreConnectScript: new("\t"),
	}}
	g.Expect(usesConnectScript(docker, blank)).To(BeFalse())
	g.Expect(usesConnectScript(kubernetes, blank)).To(BeFalse())

	blank.DockerConnectScriptIsSudo = true
	g.Expect(usesConnectScript(docker, blank)).To(BeTrue())
	g.Expect(usesConnectScript(kubernetes, blank)).To(BeFalse())
}

func TestRenderConnectScriptKubernetes(t *testing.T) {
	g := NewWithT(t)
	target := types.DeploymentTarget{Type: types.DeploymentTypeKubernetes, Namespace: new("it's")}

	org := orgWithScripts(types.FeaturePrePostScripts)

	script, err := renderConnectScript(target, org, "https://distr.example/connect")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(script).To(Equal(`#!/bin/sh

export DISTR_NAMESPACE='it'\''s'

# Pre-connect script
check-kubernetes

# Connect to Distr controller
kubectl apply -n 'it'\''s' -f 'https://distr.example/connect'

# Post-connect script
after-kubernetes
`))
}
