package kubecollect

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	chart "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/release/common"
	rspb "helm.sh/helm/v4/pkg/release/v1"
	"helm.sh/helm/v4/pkg/storage/driver"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCollectHelmReleases(t *testing.T) {
	g := NewWithT(t)
	client := fake.NewClientset()
	secrets := driver.NewSecrets(client.CoreV1().Secrets("shop"))
	deployed := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	for _, release := range []*rspb.Release{
		{
			Name: "web", Namespace: "shop", Version: 2,
			Info: &rspb.Info{
				FirstDeployed: deployed, LastDeployed: deployed.Add(time.Hour),
				Status: common.StatusDeployed, Description: "Upgrade complete",
			},
			Chart:  &chart.Chart{Metadata: &chart.Metadata{Name: "web", Version: "0.2.0", AppVersion: "1.2.0"}},
			Config: map[string]any{"replicas": 2, "database": map[string]any{"host": "db"}},
		},
		{
			Name: "web", Namespace: "shop", Version: 1,
			Info: &rspb.Info{
				FirstDeployed: deployed, LastDeployed: deployed,
				Status: common.StatusSuperseded, Description: "Install complete",
			},
			Chart:  &chart.Chart{Metadata: &chart.Metadata{Name: "web", Version: "0.1.0", AppVersion: "1.1.0"}},
			Config: map[string]any{"replicas": 1},
		},
	} {
		g.Expect(secrets.Create("sh.helm.release.v1.web.v"+strconv.Itoa(release.Version), release)).To(Succeed())
	}

	dir := t.TempDir()
	c := Collector{Clients: &Clients{Kubernetes: client}, Options: DefaultOptions(), Sink: DirSink{Dir: dir}}
	g.Expect(c.collectHelmReleases(t.Context(), "shop", true)).To(Succeed())
	g.Expect(c.problems).To(BeEmpty())

	release, err := os.ReadFile(filepath.Join(dir, "namespaces", "shop", "helm", "web", "release.yaml"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(release).To(MatchYAML(`
name: web
chart: web
chartVersion: 0.2.0
appVersion: 1.2.0
revision: 2
status: deployed
firstDeployed: "2026-10-01T10:00:00Z"
lastDeployed: "2026-10-01T11:00:00Z"
description: Upgrade complete
history:
  - revision: 1
    status: superseded
    updated: "2026-10-01T10:00:00Z"
    chart: web
    chartVersion: 0.1.0
    appVersion: 1.1.0
    description: Install complete
  - revision: 2
    status: deployed
    updated: "2026-10-01T11:00:00Z"
    chart: web
    chartVersion: 0.2.0
    appVersion: 1.2.0
    description: Upgrade complete
`))

	values, err := os.ReadFile(filepath.Join(dir, "namespaces", "shop", "helm", "web", "values.yaml"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(values).To(MatchYAML("replicas: 2\ndatabase:\n  host: db\n"))
}
