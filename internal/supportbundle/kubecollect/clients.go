package kubecollect

import (
	"errors"
	"fmt"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeContext is the current context of the customer's kubeconfig, which is everything the collector
// connects with. Nothing is installed in the cluster.
type KubeContext struct {
	Name      string
	Server    string
	Namespace string
	Config    *rest.Config
}

func LoadKubeContext() (*KubeContext, error) {
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{},
	)
	raw, err := clientConfig.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read the kubeconfig: %w", err)
	}
	if raw.CurrentContext == "" {
		return nil, errors.New("the kubeconfig has no current context, select one with kubectl config use-context")
	}
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return nil, fmt.Errorf("failed to read the namespace of the current context: %w", err)
	}
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load the current context: %w", err)
	}
	// The collector issues hundreds of list and log requests, which the client-go defaults of 5 QPS
	// would throttle for minutes. Deprecation warnings for listed APIs are of no use to the customer.
	config.QPS = 50
	config.Burst = 100
	config.WarningHandler = rest.NoWarnings{}

	result := KubeContext{Name: raw.CurrentContext, Namespace: namespace, Config: config}
	if kubeContext, ok := raw.Contexts[raw.CurrentContext]; ok {
		if cluster, ok := raw.Clusters[kubeContext.Cluster]; ok {
			result.Server = cluster.Server
		}
	}
	return &result, nil
}

type Clients struct {
	Kubernetes kubernetes.Interface
	Dynamic    dynamic.Interface
	Discovery  discovery.DiscoveryInterface
	Metadata   metadata.Interface
}

func NewClients(config *rest.Config) (*Clients, error) {
	if k8s, err := kubernetes.NewForConfig(config); err != nil {
		return nil, err
	} else if dyn, err := dynamic.NewForConfig(config); err != nil {
		return nil, err
	} else if meta, err := metadata.NewForConfig(config); err != nil {
		return nil, err
	} else {
		return &Clients{Kubernetes: k8s, Dynamic: dyn, Discovery: k8s.Discovery(), Metadata: meta}, nil
	}
}
