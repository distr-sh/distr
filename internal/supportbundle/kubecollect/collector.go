package kubecollect

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	listPageSize = 500
	// maxLogReadBytes bounds what is read of a single log before it is cut to MaxResourceBytes, since
	// the tail lines alone do not bound the size of a log with very long lines.
	maxLogReadBytes = 64 * 1024 * 1024
)

var (
	nodesGVR      = schema.GroupVersionResource{Version: "v1", Resource: "nodes"}
	namespacesGVR = schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}
	eventsGVR     = schema.GroupVersionResource{Version: "v1", Resource: "events"}
	secretsGVR    = schema.GroupVersionResource{Version: "v1", Resource: "secrets"}
	crdsGVR       = schema.GroupVersionResource{
		Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions",
	}
)

type Collector struct {
	Clients  *Clients
	Context  *KubeContext
	Options  Options
	Sink     Sink
	Progress io.Writer

	collected int
	problems  []string
}

type Result struct {
	// Collected is the number of files with collected data, not counting the summary and the
	// collection errors.
	Collected int
	Problems  int
}

type summary struct {
	CollectorVersion string          `json:"collectorVersion"`
	CollectedAt      time.Time       `json:"collectedAt"`
	Context          string          `json:"context"`
	Namespaces       []string        `json:"namespaces"`
	Collected        Groups          `json:"collected"`
	Options          Options         `json:"options"`
	Cluster          *clusterSummary `json:"cluster,omitempty"`
}

type clusterSummary struct {
	Server            string `json:"server,omitempty"`
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
	Distribution      string `json:"distribution"`
}

type crdSummary struct {
	Name     string   `json:"name"`
	Group    string   `json:"group"`
	Kind     string   `json:"kind"`
	Scope    string   `json:"scope"`
	Versions []string `json:"versions"`
}

// Run collects the bundle into the sink. Everything the cluster refuses or fails to return is
// recorded in collection-errors.txt and the run continues. Only a failure of the sink aborts it.
func (c *Collector) Run(ctx context.Context, namespaces []string, groups Groups) (Result, error) {
	s := summary{
		CollectorVersion: buildconfig.Version(),
		CollectedAt:      time.Now().UTC(),
		Context:          c.Context.Name,
		Namespaces:       namespaces,
		Collected:        groups,
		Options:          c.Options,
	}

	if groups.Cluster {
		c.progressf("Collecting cluster information")
		if cluster, err := c.collectCluster(ctx); err != nil {
			return Result{}, err
		} else {
			s.Cluster = cluster
		}
	}

	var resources []schema.GroupVersionResource
	if groups.Resources {
		var err error
		if resources, err = c.namespacedResources(); err != nil {
			c.problemf("discover the API resources: %v", err)
		}
	}

	for _, namespace := range namespaces {
		c.progressf("Collecting namespace %s", namespace)
		if groups.Resources {
			if err := c.collectResources(ctx, namespace, resources); err != nil {
				return Result{}, err
			}
		}
		if groups.Events {
			if err := c.collectEvents(ctx, namespace); err != nil {
				return Result{}, err
			}
		}
		if groups.Logs {
			if err := c.collectLogs(ctx, namespace); err != nil {
				return Result{}, err
			}
		}
	}

	if content, err := yaml.Marshal(s); err != nil {
		return Result{}, err
	} else if err := c.write(ctx, "summary.yaml", types.SupportBundleResourceContentTypeYAML, content); err != nil {
		return Result{}, err
	}
	if len(c.problems) > 0 {
		content := []byte(strings.Join(c.problems, "\n") + "\n")
		if err := c.write(ctx, "collection-errors.txt", types.SupportBundleResourceContentTypeText, content); err != nil {
			return Result{}, err
		}
	}
	return Result{Collected: c.collected, Problems: len(c.problems)}, nil
}

func (c *Collector) collectCluster(ctx context.Context) (*clusterSummary, error) {
	cluster := clusterSummary{Server: c.Context.Server}

	if version, err := c.Clients.Discovery.ServerVersion(); err != nil {
		c.problemf("get the server version: %v", err)
	} else if content, err := json.MarshalIndent(version, "", "  "); err != nil {
		return nil, err
	} else if err := c.collect(
		ctx, "cluster/version.json", types.SupportBundleResourceContentTypeJSON, content,
	); err != nil {
		return nil, err
	} else {
		cluster.KubernetesVersion = version.GitVersion
	}

	var nodes []node
	if items, err := c.list(ctx, nodesGVR, ""); err != nil {
		c.problemf("list the nodes: %v", err)
	} else {
		for _, item := range items {
			providerID, _, _ := unstructured.NestedString(item.Object, "spec", "providerID")
			nodes = append(nodes, node{Name: item.GetName(), Labels: item.GetLabels(), ProviderID: providerID})
			if err := c.collectObject(ctx, "cluster/nodes/"+item.GetName()+".yaml", item); err != nil {
				return nil, err
			}
		}
	}
	cluster.Distribution = detectDistribution(cluster.KubernetesVersion, nodes)

	if items, err := c.list(ctx, namespacesGVR, ""); err != nil {
		c.problemf("list the namespaces: %v", err)
	} else if err := c.collectList(ctx, "cluster/namespaces.yaml", items); err != nil {
		return nil, err
	}

	if items, err := c.list(ctx, crdsGVR, ""); err != nil {
		c.problemf("list the custom resource definitions: %v", err)
	} else {
		crds := make([]crdSummary, 0, len(items))
		for _, item := range items {
			crd := crdSummary{Name: item.GetName()}
			crd.Group, _, _ = unstructured.NestedString(item.Object, "spec", "group")
			crd.Kind, _, _ = unstructured.NestedString(item.Object, "spec", "names", "kind")
			crd.Scope, _, _ = unstructured.NestedString(item.Object, "spec", "scope")
			versions, _, _ := unstructured.NestedSlice(item.Object, "spec", "versions")
			for _, version := range versions {
				if v, ok := version.(map[string]any); ok {
					crd.Versions = append(crd.Versions, fmt.Sprint(v["name"]))
				}
			}
			crds = append(crds, crd)
		}
		if content, err := yaml.Marshal(crds); err != nil {
			return nil, err
		} else if err := c.collect(
			ctx, "cluster/crds.yaml", types.SupportBundleResourceContentTypeYAML, content,
		); err != nil {
			return nil, err
		}
	}

	return &cluster, nil
}

func (c *Collector) collectResources(
	ctx context.Context,
	namespace string,
	resources []schema.GroupVersionResource,
) error {
	var objects, kinds int
	for _, gvr := range resources {
		items, err := c.list(ctx, gvr, namespace)
		if err != nil {
			c.problemf("list %s in namespace %s: %v", resourceDir(gvr), namespace, err)
			continue
		}
		if len(items) > 0 {
			kinds++
		}
		for _, item := range items {
			name := fmt.Sprintf("namespaces/%s/%s/%s.yaml", namespace, resourceDir(gvr), item.GetName())
			if err := c.collectObject(ctx, name, item); err != nil {
				return err
			}
			objects++
		}
	}
	c.progressf("  %d resources of %d kinds", objects, kinds)
	return nil
}

// namespacedResources is every namespaced resource the cluster serves that can be listed, custom
// resources included. Discovery returns what it found even when some API groups fail, which is
// common for aggregated APIs whose backing service is down.
func (c *Collector) namespacedResources() ([]schema.GroupVersionResource, error) {
	lists, err := c.Clients.Discovery.ServerPreferredNamespacedResources()
	var result []schema.GroupVersionResource
	for _, list := range lists {
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil || gv.Group == "metrics.k8s.io" {
			continue
		}
		for _, resource := range list.APIResources {
			if strings.Contains(resource.Name, "/") || resource.Name == "events" ||
				!slices.Contains(resource.Verbs, "list") {
				continue
			}
			result = append(result, gv.WithResource(resource.Name))
		}
	}
	slices.SortFunc(result, func(a, b schema.GroupVersionResource) int {
		return strings.Compare(resourceDir(a), resourceDir(b))
	})
	return result, err
}

func resourceDir(gvr schema.GroupVersionResource) string {
	if gvr.Group == "" {
		return gvr.Resource
	}
	return gvr.Resource + "." + gvr.Group
}

func (c *Collector) collectEvents(ctx context.Context, namespace string) error {
	items, err := c.list(ctx, eventsGVR, namespace)
	if err != nil {
		c.problemf("list the events in namespace %s: %v", namespace, err)
		return nil
	}
	slices.SortStableFunc(items, func(a, b unstructured.Unstructured) int {
		return eventTime(a).Compare(eventTime(b))
	})
	objects := make([]map[string]any, len(items))
	for i, item := range items {
		sanitize(item.Object, c.Options)
		objects[i] = item.Object
	}

	// Drop the oldest events until the newest ones fit, since a truncated YAML list would be invalid.
	for omitted := 0; ; {
		var header string
		if omitted > 0 {
			header = fmt.Sprintf(
				"# --- distr: %d older events omitted to stay within %d bytes ---\n", omitted, c.Options.MaxResourceBytes,
			)
		}
		content, err := yaml.Marshal(map[string]any{"apiVersion": "v1", "kind": "List", "items": objects[omitted:]})
		if err != nil {
			return err
		}
		if len(header)+len(content) <= c.Options.MaxResourceBytes || omitted == len(objects) {
			c.progressf("  %d events", len(objects)-omitted)
			name := "namespaces/" + namespace + "/events.yaml"
			return c.collect(ctx, name, types.SupportBundleResourceContentTypeYAML, append([]byte(header), content...))
		}
		omitted += max(1, (len(objects)-omitted)/4)
	}
}

func eventTime(event unstructured.Unstructured) time.Time {
	for _, field := range [][]string{{"lastTimestamp"}, {"eventTime"}, {"metadata", "creationTimestamp"}} {
		if value, _, _ := unstructured.NestedString(event.Object, field...); value != "" {
			if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func (c *Collector) collectLogs(ctx context.Context, namespace string) error {
	var pods []corev1.Pod
	opts := metav1.ListOptions{Limit: listPageSize}
	for {
		list, err := c.Clients.Kubernetes.CoreV1().Pods(namespace).List(ctx, opts)
		if err != nil {
			c.problemf("list the pods in namespace %s for their logs: %v", namespace, err)
			return nil
		}
		pods = append(pods, list.Items...)
		if opts.Continue = list.Continue; opts.Continue == "" {
			break
		}
	}

	var logs int
	for _, pod := range pods {
		for _, container := range podContainers(pod) {
			status := container.status
			if status == nil ||
				(status.State.Waiting != nil && status.RestartCount == 0 && status.LastTerminationState.Terminated == nil) {
				continue
			}
			base := fmt.Sprintf("namespaces/%s/pods/%s/%s", namespace, pod.Name, container.name)
			if collected, err := c.collectLog(ctx, pod, container.name, false, base+".log"); err != nil {
				return err
			} else if collected {
				logs++
			}
			if status.RestartCount > 0 || status.LastTerminationState.Terminated != nil {
				if collected, err := c.collectLog(ctx, pod, container.name, true, base+".previous.log"); err != nil {
					return err
				} else if collected {
					logs++
				}
			}
		}
	}
	c.progressf("  %d container logs of %d pods", logs, len(pods))
	return nil
}

type podContainer struct {
	name   string
	status *corev1.ContainerStatus
}

func podContainers(pod corev1.Pod) []podContainer {
	find := func(statuses []corev1.ContainerStatus, name string) *corev1.ContainerStatus {
		for i := range statuses {
			if statuses[i].Name == name {
				return &statuses[i]
			}
		}
		return nil
	}
	result := make(
		[]podContainer, 0,
		len(pod.Spec.InitContainers)+len(pod.Spec.Containers)+len(pod.Spec.EphemeralContainers),
	)
	for _, container := range pod.Spec.InitContainers {
		result = append(result, podContainer{container.Name, find(pod.Status.InitContainerStatuses, container.Name)})
	}
	for _, container := range pod.Spec.Containers {
		result = append(result, podContainer{container.Name, find(pod.Status.ContainerStatuses, container.Name)})
	}
	for _, container := range pod.Spec.EphemeralContainers {
		result = append(result, podContainer{container.Name, find(pod.Status.EphemeralContainerStatuses, container.Name)})
	}
	return result
}

func (c *Collector) collectLog(
	ctx context.Context,
	pod corev1.Pod,
	container string,
	previous bool,
	name string,
) (bool, error) {
	stream, err := c.Clients.Kubernetes.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: container,
		Previous:  previous,
		TailLines: &c.Options.LogTailLines,
	}).Stream(ctx)
	if err != nil {
		c.problemf("get the logs of %s: %v", name, err)
		return false, nil
	}
	defer stream.Close()
	content, err := io.ReadAll(io.LimitReader(stream, maxLogReadBytes))
	if err != nil {
		c.problemf("read the logs of %s: %v", name, err)
		return false, nil
	}
	return true, c.collect(ctx, name, types.SupportBundleResourceContentTypeLog, content)
}

func (c *Collector) list(
	ctx context.Context,
	gvr schema.GroupVersionResource,
	namespace string,
) ([]unstructured.Unstructured, error) {
	var items []unstructured.Unstructured
	opts := metav1.ListOptions{Limit: listPageSize}
	for {
		list, err := c.Clients.Dynamic.Resource(gvr).Namespace(namespace).List(ctx, opts)
		if err != nil {
			return nil, err
		}
		items = append(items, list.Items...)
		if opts.Continue = list.GetContinue(); opts.Continue == "" {
			return items, nil
		}
	}
}

func (c *Collector) collectObject(ctx context.Context, name string, item unstructured.Unstructured) error {
	sanitize(item.Object, c.Options)
	if content, err := yaml.Marshal(item.Object); err != nil {
		return err
	} else {
		return c.collect(ctx, name, types.SupportBundleResourceContentTypeYAML, content)
	}
}

func (c *Collector) collectList(ctx context.Context, name string, items []unstructured.Unstructured) error {
	objects := make([]map[string]any, len(items))
	for i, item := range items {
		sanitize(item.Object, c.Options)
		objects[i] = item.Object
	}
	if content, err := yaml.Marshal(map[string]any{"apiVersion": "v1", "kind": "List", "items": objects}); err != nil {
		return err
	} else {
		return c.collect(ctx, name, types.SupportBundleResourceContentTypeYAML, content)
	}
}

func (c *Collector) collect(
	ctx context.Context,
	name string,
	contentType types.SupportBundleResourceContentType,
	content []byte,
) error {
	if err := c.write(ctx, name, contentType, content); err != nil {
		return err
	}
	c.collected++
	return nil
}

func (c *Collector) write(
	ctx context.Context,
	name string,
	contentType types.SupportBundleResourceContentType,
	content []byte,
) error {
	if contentType == types.SupportBundleResourceContentTypeLog {
		content, _ = keepTail(content, c.Options.MaxResourceBytes)
	} else if cut, ok := keepHead(content, c.Options.MaxResourceBytes); ok {
		c.problemf("%s is larger than %d bytes and was cut", name, c.Options.MaxResourceBytes)
		content = cut
	}
	if err := c.Sink.Write(ctx, name, contentType, content); err != nil {
		return fmt.Errorf("failed to write %s: %w", name, err)
	}
	return nil
}

func (c *Collector) problemf(format string, args ...any) {
	c.problems = append(c.problems, fmt.Sprintf(format, args...))
}

func (c *Collector) progressf(format string, args ...any) {
	if c.Progress != nil {
		_, _ = fmt.Fprintf(c.Progress, format+"\n", args...)
	}
}
