package supportcollector

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/distr-sh/distr/internal/contenttype"
	"helm.sh/helm/v4/pkg/release"
	rspb "helm.sh/helm/v4/pkg/release/v1"
	"helm.sh/helm/v4/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

const helmReleaseSecretType corev1.SecretType = "helm.sh/release.v1"

type helmReleaseSummary struct {
	Name          string                `json:"name"`
	Chart         string                `json:"chart"`
	ChartVersion  string                `json:"chartVersion"`
	AppVersion    string                `json:"appVersion,omitempty"`
	Revision      int                   `json:"revision"`
	Status        string                `json:"status"`
	FirstDeployed string                `json:"firstDeployed,omitempty"`
	LastDeployed  string                `json:"lastDeployed,omitempty"`
	Description   string                `json:"description,omitempty"`
	History       []helmRevisionSummary `json:"history"`
}

type helmRevisionSummary struct {
	Revision     int    `json:"revision"`
	Status       string `json:"status"`
	Updated      string `json:"updated,omitempty"`
	Chart        string `json:"chart"`
	ChartVersion string `json:"chartVersion"`
	AppVersion   string `json:"appVersion,omitempty"`
	Description  string `json:"description,omitempty"`
}

// collectHelmReleases writes what `helm status` and `helm history` show for every release in the
// namespace and, with values, what `helm get values` shows for its latest revision.
func (c *Collector) collectHelmReleases(ctx context.Context, namespace string, values bool) error {
	// The driver reads all release Secrets of the namespace in a single request without paging, the
	// same way `helm list` does. It skips a Secret it cannot decode without returning an error.
	stored, err := driver.NewSecrets(c.Clients.Kubernetes.CoreV1().Secrets(namespace)).
		List(func(release.Releaser) bool { return true })
	if err != nil {
		c.problemf("list the Helm releases in namespace %s: %v", namespace, err)
		return nil
	}

	latest := map[string]*rspb.Release{}
	history := map[string][]helmRevisionSummary{}
	for _, releaser := range stored {
		release, ok := releaser.(*rspb.Release)
		if !ok || release.Info == nil || release.Chart == nil || release.Chart.Metadata == nil {
			c.problemf("decode a Helm release in namespace %s: unexpected content", namespace)
			continue
		}
		history[release.Name] = append(history[release.Name], helmRevisionSummary{
			Revision:     release.Version,
			Status:       release.Info.Status.String(),
			Updated:      formatHelmTime(release.Info.LastDeployed),
			Chart:        release.Chart.Metadata.Name,
			ChartVersion: release.Chart.Metadata.Version,
			AppVersion:   release.Chart.Metadata.AppVersion,
			Description:  release.Info.Description,
		})
		if current, ok := latest[release.Name]; !ok || release.Version > current.Version {
			latest[release.Name] = release
		}
	}

	for _, name := range slices.Sorted(maps.Keys(latest)) {
		release := latest[name]
		revisions := history[name]
		slices.SortFunc(revisions, func(a, b helmRevisionSummary) int { return a.Revision - b.Revision })
		dir := fmt.Sprintf("namespaces/%s/helm/%s/", namespace, name)
		if content, err := yaml.Marshal(helmReleaseSummary{
			Name:          release.Name,
			Chart:         release.Chart.Metadata.Name,
			ChartVersion:  release.Chart.Metadata.Version,
			AppVersion:    release.Chart.Metadata.AppVersion,
			Revision:      release.Version,
			Status:        release.Info.Status.String(),
			FirstDeployed: formatHelmTime(release.Info.FirstDeployed),
			LastDeployed:  formatHelmTime(release.Info.LastDeployed),
			Description:   release.Info.Description,
			History:       revisions,
		}); err != nil {
			return err
		} else if err := c.collect(ctx, dir+"release.yaml", contenttype.MediaTypeYAML, content); err != nil {
			return err
		}
		if values {
			if content, err := yaml.Marshal(release.Config); err != nil {
				return err
			} else if err := c.collect(ctx, dir+"values.yaml", contenttype.MediaTypeYAML, content); err != nil {
				return err
			}
		}
	}
	c.progressf("  %d Helm releases", len(latest))
	return nil
}

func formatHelmTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}
