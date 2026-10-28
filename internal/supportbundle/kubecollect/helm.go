package kubecollect

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/distr-sh/distr/internal/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const helmReleaseSecretType corev1.SecretType = "helm.sh/release.v1"

// helmListPageSize is small because every release Secret holds the whole chart and the rendered
// manifest, up to the 1 MiB a Secret may hold.
const helmListPageSize = 10

// helmStoredRelease is the part of the release Helm stores as gzipped JSON, the same in Helm 3 and
// Helm 4. The timestamps stay strings because Helm 3 writes an empty string for one that is unset.
type helmStoredRelease struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
	Info    struct {
		FirstDeployed string `json:"first_deployed"`
		LastDeployed  string `json:"last_deployed"`
		Description   string `json:"description"`
		Status        string `json:"status"`
	} `json:"info"`
	Chart struct {
		Metadata struct {
			Name       string `json:"name"`
			Version    string `json:"version"`
			AppVersion string `json:"appVersion"`
		} `json:"metadata"`
	} `json:"chart"`
	Config map[string]any `json:"config"`
}

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
	latest := map[string]*helmStoredRelease{}
	history := map[string][]helmRevisionSummary{}
	opts := metav1.ListOptions{LabelSelector: "owner=helm", Limit: helmListPageSize}
	for {
		list, err := c.Clients.Kubernetes.CoreV1().Secrets(namespace).List(ctx, opts)
		if err != nil {
			c.problemf("list the Helm releases in namespace %s: %v", namespace, err)
			return nil
		}
		for _, secret := range list.Items {
			if secret.Type != helmReleaseSecretType {
				continue
			}
			release, err := decodeHelmRelease(secret.Data["release"])
			if err != nil {
				c.problemf("decode the Helm release in Secret %s/%s: %v", namespace, secret.Name, err)
				continue
			}
			history[release.Name] = append(history[release.Name], helmRevisionSummary{
				Revision:     release.Version,
				Status:       release.Info.Status,
				Updated:      release.Info.LastDeployed,
				Chart:        release.Chart.Metadata.Name,
				ChartVersion: release.Chart.Metadata.Version,
				AppVersion:   release.Chart.Metadata.AppVersion,
				Description:  release.Info.Description,
			})
			if current, ok := latest[release.Name]; !ok || release.Version > current.Version {
				latest[release.Name] = release
			}
		}
		if opts.Continue = list.Continue; opts.Continue == "" {
			break
		}
	}

	names := make([]string, 0, len(latest))
	for name := range latest {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
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
			Status:        release.Info.Status,
			FirstDeployed: release.Info.FirstDeployed,
			LastDeployed:  release.Info.LastDeployed,
			Description:   release.Info.Description,
			History:       revisions,
		}); err != nil {
			return err
		} else if err := c.collect(ctx, dir+"release.yaml", types.SupportBundleResourceContentTypeYAML, content); err != nil {
			return err
		}
		if values {
			if content, err := yaml.Marshal(release.Config); err != nil {
				return err
			} else if err := c.collect(ctx, dir+"values.yaml", types.SupportBundleResourceContentTypeYAML, content); err != nil {
				return err
			}
		}
	}
	c.progressf("  %d Helm releases", len(names))
	return nil
}

// decodeHelmRelease reverses what Helm stores in the release key of the Secret: JSON, gzipped and
// base64 encoded once more on top of the encoding of Secret data.
func decodeHelmRelease(data []byte) (*helmStoredRelease, error) {
	decoded, err := base64.StdEncoding.AppendDecode(nil, data)
	if err != nil {
		return nil, err
	}
	if bytes.HasPrefix(decoded, []byte{0x1f, 0x8b, 0x08}) {
		reader, err := gzip.NewReader(bytes.NewReader(decoded))
		if err != nil {
			return nil, err
		}
		if decoded, err = io.ReadAll(reader); err != nil {
			return nil, err
		}
	}
	var release helmStoredRelease
	if err := json.Unmarshal(decoded, &release); err != nil {
		return nil, err
	}
	return &release, nil
}
