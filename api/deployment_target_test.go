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
