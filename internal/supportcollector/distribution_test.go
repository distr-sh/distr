package supportcollector

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestDetectDistribution(t *testing.T) {
	for name, tc := range map[string]struct {
		gitVersion string
		nodes      []node
		expected   Distribution
	}{
		"k3s":         {gitVersion: "v1.33.4+k3s1", expected: DistributionK3s},
		"eks version": {gitVersion: "v1.32.9-eks-113cf36", expected: DistributionEKS},
		"eks nodes": {
			gitVersion: "v1.32.9",
			nodes:      []node{{Labels: map[string]string{"eks.amazonaws.com/nodegroup": "default"}}},
			expected:   DistributionEKS,
		},
		"version wins over nodes": {
			gitVersion: "v1.33.4+rke2r1",
			nodes:      []node{{Labels: map[string]string{"minikube.k8s.io/name": "minikube"}}},
			expected:   DistributionRKE2,
		},
		"minikube": {
			gitVersion: "v1.34.0",
			nodes:      []node{{Name: "minikube", Labels: map[string]string{"minikube.k8s.io/name": "minikube"}}},
			expected:   DistributionMinikube,
		},
		"kind": {
			gitVersion: "v1.34.0",
			nodes:      []node{{Name: "kind-control-plane", ProviderID: "kind://docker/kind/kind-control-plane"}},
			expected:   DistributionKind,
		},
		"unknown": {
			gitVersion: "v1.34.0",
			nodes:      []node{{Name: "node-1", Labels: map[string]string{"kubernetes.io/os": "linux"}}},
			expected:   DistributionUnknown,
		},
	} {
		t.Run(name, func(t *testing.T) {
			NewWithT(t).Expect(detectDistribution(tc.gitVersion, tc.nodes)).To(Equal(tc.expected))
		})
	}
}
