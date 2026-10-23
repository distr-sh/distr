package kubecollect

import (
	"strings"
)

type node struct {
	Name       string
	Labels     map[string]string
	ProviderID string
}

// detectDistribution guesses the Kubernetes distribution from the markers the distributions leave in
// the server version and on their nodes. It is a hint for the vendor, so an unknown cluster is fine.
func detectDistribution(gitVersion string, nodes []node) string {
	switch {
	case strings.Contains(gitVersion, "+k3s"):
		return "k3s"
	case strings.Contains(gitVersion, "+rke2"):
		return "rke2"
	case strings.Contains(gitVersion, "-eks-"):
		return "eks"
	case strings.Contains(gitVersion, "-gke."):
		return "gke"
	}
	for _, n := range nodes {
		switch {
		case n.Labels["node.openshift.io/os_id"] != "":
			return "openshift"
		case n.Labels["minikube.k8s.io/name"] != "":
			return "minikube"
		case n.Labels["kubernetes.azure.com/cluster"] != "":
			return "aks"
		case n.Labels["eks.amazonaws.com/nodegroup"] != "":
			return "eks"
		case strings.HasPrefix(n.ProviderID, "kind://"):
			return "kind"
		case strings.HasPrefix(n.ProviderID, "gce://"):
			return "gke"
		case n.Name == "docker-desktop":
			return "docker-desktop"
		}
	}
	return "unknown"
}
