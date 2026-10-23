package kubecollect

import (
	"context"
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	releases := map[string][]string{}
	add := func(namespace string) bool {
		list, err := clients.Metadata.Resource(secretsGVR).Namespace(namespace).
			List(ctx, metav1.ListOptions{LabelSelector: "owner=helm"})
		if err != nil {
			return false
		}
		for _, secret := range list.Items {
			if name := secret.Labels["name"]; name != "" && !slices.Contains(releases[secret.Namespace], name) {
				releases[secret.Namespace] = append(releases[secret.Namespace], name)
			}
		}
		return true
	}
	if !add(metav1.NamespaceAll) {
		for _, option := range options {
			add(option.Name)
		}
	}
	for i := range options {
		options[i].HelmReleases = releases[options[i].Name]
		slices.Sort(options[i].HelmReleases)
	}
}
