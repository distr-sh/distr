package kubecollect

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// KubeContext is the context of the customer's kubeconfig the collector connects with, the one
// --context names or the customer selects. Nothing is installed in the cluster.
type KubeContext struct {
	Name      string
	Server    string
	Namespace string
	Config    *rest.Config
}

type KubeContextOption struct {
	Name   string
	Server string
}

// ListKubeContexts returns the contexts of the kubeconfig sorted by name and the current one.
func ListKubeContexts(configFlags *genericclioptions.ConfigFlags) ([]KubeContextOption, string, error) {
	// ConfigFlags keeps the loader of its first use, which has the --context of that moment, so the
	// kubeconfig is read here without it to let the selected context take effect afterwards.
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if configFlags.KubeConfig != nil {
		rules.ExplicitPath = *configFlags.KubeConfig
	}
	raw, err := rules.Load()
	if err != nil {
		return nil, "", fmt.Errorf("failed to read the kubeconfig: %w", err)
	}
	options := make([]KubeContextOption, 0, len(raw.Contexts))
	for _, name := range slices.Sorted(maps.Keys(raw.Contexts)) {
		option := KubeContextOption{Name: name}
		if cluster, ok := raw.Clusters[raw.Contexts[name].Cluster]; ok {
			option.Server = cluster.Server
		}
		options = append(options, option)
	}
	return options, raw.CurrentContext, nil
}

func LoadKubeContext(configFlags *genericclioptions.ConfigFlags) (*KubeContext, error) {
	clientConfig := configFlags.ToRawKubeConfigLoader()
	raw, err := clientConfig.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read the kubeconfig: %w", err)
	}
	// RawConfig is the kubeconfig as it is on disk, without the --context override.
	name := raw.CurrentContext
	if configFlags.Context != nil && *configFlags.Context != "" {
		name = *configFlags.Context
	}
	if name == "" {
		return nil, errors.New("the kubeconfig has no context, select another one with --kubeconfig")
	} else if _, ok := raw.Contexts[name]; !ok {
		return nil, fmt.Errorf("the kubeconfig has no context named %q", name)
	}
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return nil, fmt.Errorf("failed to read the namespace of the context: %w", err)
	}
	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load the context: %w", err)
	}
	// The collector issues hundreds of list and log requests, which the client-go defaults of 5 QPS
	// would throttle for minutes. Deprecation warnings for listed APIs are of no use to the customer.
	config.QPS = 50
	config.Burst = 100
	config.WarningHandler = rest.NoWarnings{}

	return &KubeContext{Name: name, Server: config.Host, Namespace: namespace, Config: config}, nil
}

// WriteKubeconfig writes a kubeconfig holding only the collector's context, with its namespace and
// with the files it references inlined, so that kubectl and helm run afterwards reach the same
// cluster and namespace whatever the current context of the customer's kubeconfig is.
func WriteKubeconfig(configFlags *genericclioptions.ConfigFlags, kubeContext *KubeContext, path string) error {
	raw, err := configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return fmt.Errorf("failed to read the kubeconfig: %w", err)
	}
	config := raw.DeepCopy()
	config.CurrentContext = kubeContext.Name
	if err := clientcmdapi.MinifyConfig(config); err != nil {
		return fmt.Errorf("failed to reduce the kubeconfig to the context: %w", err)
	} else if err := clientcmdapi.FlattenConfig(config); err != nil {
		return fmt.Errorf("failed to inline the files the kubeconfig references: %w", err)
	}
	config.Contexts[kubeContext.Name].Namespace = kubeContext.Namespace
	if err := clientcmd.WriteToFile(*config, path); err != nil {
		return fmt.Errorf("failed to write the kubeconfig: %w", err)
	}
	return nil
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
