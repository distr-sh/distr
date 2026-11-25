package api_test

import (
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func TestCreateDeploymentTargetRequestValidateDockerRejectsKubernetesFields(t *testing.T) {
	invalid := map[string]api.CreateDeploymentTargetRequest{
		"namespace":       {Type: types.DeploymentTypeDocker, Namespace: new("default")},
		"empty namespace": {Type: types.DeploymentTypeDocker, Namespace: new("")},
		"scope":           {Type: types.DeploymentTypeDocker, Scope: new(types.DeploymentTargetScopeNamespace)},
	}
	for name, request := range invalid {
		t.Run(name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(request.Validate()).To(HaveOccurred())
		})
	}

	g := NewWithT(t)
	g.Expect((&api.CreateDeploymentTargetRequest{Type: types.DeploymentTypeDocker}).Validate()).To(Succeed())
}

func TestDeploymentTargetSettingsValidatePodScheduling(t *testing.T) {
	valid := api.DeploymentTargetSettings{
		PodAnnotations: map[string]string{"example.com/do-not-disrupt": "true"},
		NodeSelector:   map[string]string{"example.com/pool": "system"},
		Tolerations: []api.DeploymentTargetToleration{
			{Key: "CriticalAddonsOnly", Operator: "Exists", Effect: "NoSchedule"},
			{Key: "example.com/dedicated", Value: "yes"},
			{Operator: "Exists"},
		},
	}
	g := NewWithT(t)
	g.Expect(valid.Validate(types.DeploymentTypeKubernetes)).To(Succeed())
	g.Expect(valid.Validate(types.DeploymentTypeDocker)).To(HaveOccurred())

	invalid := map[string]api.DeploymentTargetSettings{
		"annotation key":      {PodAnnotations: map[string]string{"not a key": "x"}},
		"node selector value": {NodeSelector: map[string]string{"pool": "not a value"}},
		"exists with value":   {Tolerations: []api.DeploymentTargetToleration{{Key: "a", Operator: "Exists", Value: "b"}}},
		"equal without key":   {Tolerations: []api.DeploymentTargetToleration{{Value: "b"}}},
		"unknown operator":    {Tolerations: []api.DeploymentTargetToleration{{Key: "a", Operator: "In"}}},
		"unknown effect":      {Tolerations: []api.DeploymentTargetToleration{{Key: "a", Effect: "NoRun"}}},
	}
	for name, settings := range invalid {
		t.Run(name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(settings.Validate(types.DeploymentTypeKubernetes)).To(HaveOccurred())
		})
	}
}
