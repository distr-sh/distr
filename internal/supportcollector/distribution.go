package supportcollector

import (
	"strings"
)

type Distribution string

const (
	DistributionK3s           Distribution = "k3s"
	DistributionRKE2          Distribution = "rke2"
	DistributionEKS           Distribution = "eks"
	DistributionGKE           Distribution = "gke"
	DistributionAKS           Distribution = "aks"
	DistributionOpenShift     Distribution = "openshift"
	DistributionMinikube      Distribution = "minikube"
	DistributionKind          Distribution = "kind"
	DistributionDockerDesktop Distribution = "docker-desktop"
	DistributionUnknown       Distribution = "unknown"
)

type node struct {
	Name       string
	Labels     map[string]string
	ProviderID string
}

// detectDistribution guesses the Kubernetes distribution from the markers the distributions leave in
// the server version and on their nodes. It is a hint for the vendor, so an unknown cluster is fine.
func detectDistribution(gitVersion string, nodes []node) Distribution {
	switch {
	case strings.Contains(gitVersion, "+k3s"):
		return DistributionK3s
	case strings.Contains(gitVersion, "+rke2"):
		return DistributionRKE2
	case strings.Contains(gitVersion, "-eks-"):
		return DistributionEKS
	case strings.Contains(gitVersion, "-gke."):
		return DistributionGKE
	}
	for _, n := range nodes {
		switch {
		case n.Labels["node.openshift.io/os_id"] != "":
			return DistributionOpenShift
		case n.Labels["minikube.k8s.io/name"] != "":
			return DistributionMinikube
		case n.Labels["kubernetes.azure.com/cluster"] != "":
			return DistributionAKS
		case n.Labels["eks.amazonaws.com/nodegroup"] != "":
			return DistributionEKS
		case strings.HasPrefix(n.ProviderID, "kind://"):
			return DistributionKind
		case strings.HasPrefix(n.ProviderID, "gce://"):
			return DistributionGKE
		case n.Name == "docker-desktop":
			return DistributionDockerDesktop
		}
	}
	return DistributionUnknown
}
