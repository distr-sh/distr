package kubecollect

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestDetectDistribution(t *testing.T) {
	for name, tc := range map[string]struct {
		gitVersion string
		nodes      []node
		expected   string
	}{
		"k3s":         {gitVersion: "v1.33.4+k3s1", expected: "k3s"},
		"eks version": {gitVersion: "v1.32.9-eks-113cf36", expected: "eks"},
		"eks nodes": {
			gitVersion: "v1.32.9",
			nodes:      []node{{Labels: map[string]string{"eks.amazonaws.com/nodegroup": "default"}}},
			expected:   "eks",
		},
		"version wins over nodes": {
			gitVersion: "v1.33.4+rke2r1",
			nodes:      []node{{Labels: map[string]string{"minikube.k8s.io/name": "minikube"}}},
			expected:   "rke2",
		},
		"minikube": {
			gitVersion: "v1.34.0",
			nodes:      []node{{Name: "minikube", Labels: map[string]string{"minikube.k8s.io/name": "minikube"}}},
			expected:   "minikube",
		},
		"kind": {
			gitVersion: "v1.34.0",
			nodes:      []node{{Name: "kind-control-plane", ProviderID: "kind://docker/kind/kind-control-plane"}},
			expected:   "kind",
		},
		"unknown": {
			gitVersion: "v1.34.0",
			nodes:      []node{{Name: "node-1", Labels: map[string]string{"kubernetes.io/os": "linux"}}},
			expected:   "unknown",
		},
	} {
		t.Run(name, func(t *testing.T) {
			NewWithT(t).Expect(detectDistribution(tc.gitVersion, tc.nodes)).To(Equal(tc.expected))
		})
	}
}
