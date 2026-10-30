package kubecollect

import (
	"context"
	"slices"
	"sync"

	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const helmReleaseLookupConcurrency = 8

type NamespaceOption struct {
	Name         string
	HelmReleases []string
}

func ListNamespaces(ctx context.Context, clients *Clients) ([]NamespaceOption, error) {
	list, err := clients.Kubernetes.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	options := make([]NamespaceOption, len(list.Items))
	for i, namespace := range list.Items {
		options[i] = NamespaceOption{Name: namespace.Name}
	}
	return options, nil
}

// AddHelmReleases adds the names of the Helm releases to the namespaces they are installed in. Helm
// stores every revision of a release as a Secret labeled with the release name, so listing only
// their metadata finds the releases without downloading any Secret data. The names are only a hint,
// so a namespace the customer may not list Secrets in simply shows none.
func AddHelmReleases(ctx context.Context, clients *Clients, options []NamespaceOption) {
	var mu sync.Mutex
	releases := map[string][]string{}
	add := func(namespace string) bool {
		list, err := clients.Metadata.Resource(secretsGVR).Namespace(namespace).
			List(ctx, metav1.ListOptions{LabelSelector: "owner=helm"})
		if err != nil {
			return false
		}
		mu.Lock()
		defer mu.Unlock()
		for _, secret := range list.Items {
			if name := secret.Labels["name"]; name != "" && !slices.Contains(releases[secret.Namespace], name) {
				releases[secret.Namespace] = append(releases[secret.Namespace], name)
			}
		}
		return true
	}
	if !add(metav1.NamespaceAll) {
		// Someone who may not list Secrets cluster-wide is often refused in most namespaces too, and
		// the customer waits for these lists before the namespace prompt appears.
		var group errgroup.Group
		group.SetLimit(helmReleaseLookupConcurrency)
		for _, option := range options {
			group.Go(func() error {
				add(option.Name)
				return nil
			})
		}
		_ = group.Wait()
	}
	for i := range options {
		options[i].HelmReleases = releases[options[i].Name]
		slices.Sort(options[i].HelmReleases)
	}
}
