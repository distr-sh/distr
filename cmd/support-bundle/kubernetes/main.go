package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/distr-sh/distr/internal/supportbundle/kubecollect"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
)

const (
	uploadURLEnv    = "DISTR_SUPPORT_BUNDLE_URL"
	uploadSecretEnv = "DISTR_SUPPORT_BUNDLE_SECRET"
)

func main() {
	// client-go logs through klog, for example when it retries a request, which would interleave with
	// the prompts. Everything the customer needs to know ends up in collection-errors.txt.
	klog.LogToStderr(false)
	klog.SetOutput(io.Discard)

	configFlags := genericclioptions.NewConfigFlags(true)
	configFlags.AddFlags(pflag.CommandLine)
	opts := kubecollect.DefaultOptions()
	output := pflag.String("output", "", "directory to write the bundle into, and to upload it from when "+
		uploadURLEnv+" and "+uploadSecretEnv+" are set")
	pflag.Int64Var(&opts.LogTailLines, "log-tail-lines", opts.LogTailLines,
		"number of lines collected from the end of every container log")
	pflag.IntVar(&opts.MaxResourceBytes, "max-resource-bytes", opts.MaxResourceBytes,
		"maximum size of every file of the bundle")
	pflag.BoolVar(&opts.IncludeConfigMapData, "include-configmap-data", opts.IncludeConfigMapData,
		"collect the values of ConfigMaps")
	pflag.BoolVar(&opts.IncludeSecretData, "include-secret-data", opts.IncludeSecretData,
		"collect the decoded values of Secrets, except service account and bootstrap tokens and Helm releases")
	pflag.Parse()

	if *output == "" {
		usageError("--output is required")
	} else if pflag.NArg() > 0 {
		usageError(fmt.Sprintf("unexpected arguments: %v", pflag.Args()))
	}
	uploader, err := uploaderFromEnv()
	if err != nil {
		usageError(err.Error())
	}

	// Ctrl+C keeps the default handling and ends the process: the prompts block on the terminal
	// without a context, and the collector records a canceled request as a problem and continues.
	// The collect script owns the output directory and removes it when it is interrupted.
	if err := run(context.Background(), configFlags, *output, opts, uploader); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func usageError(message string) {
	fmt.Fprintln(os.Stderr, message)
	pflag.Usage()
	os.Exit(2)
}

// uploaderFromEnv takes the bundle from the environment rather than from flags, so that the secret
// does not show up in the process list.
func uploaderFromEnv() (*kubecollect.Uploader, error) {
	uploadURL, secret := os.Getenv(uploadURLEnv), os.Getenv(uploadSecretEnv)
	if uploadURL == "" && secret == "" {
		return nil, nil
	} else if uploadURL == "" || secret == "" {
		return nil, fmt.Errorf("%s and %s must be set together", uploadURLEnv, uploadSecretEnv)
	}
	return &kubecollect.Uploader{URL: uploadURL, Secret: secret, Client: http.DefaultClient}, nil
}

func run(
	ctx context.Context,
	configFlags *genericclioptions.ConfigFlags,
	output string,
	opts kubecollect.Options,
	uploader *kubecollect.Uploader,
) error {
	kubeContext, err := kubecollect.LoadKubeContext(configFlags)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Context:   %s\nCluster:   %s\nNamespace: %s\n\n",
		kubeContext.Name, kubeContext.Server, kubeContext.Namespace)

	clients, err := kubecollect.NewClients(kubeContext.Config)
	if err != nil {
		return err
	}

	namespaces, err := kubecollect.ListNamespaces(ctx, clients)
	if err != nil {
		fmt.Fprintf(os.Stderr, "The namespaces of the cluster cannot be listed (%v),\n", err)
		fmt.Fprintf(os.Stderr, "so only the namespace of the context is offered.\n\n")
		namespaces = []kubecollect.NamespaceOption{{Name: kubeContext.Namespace}}
	}
	kubecollect.AddHelmReleases(ctx, clients, namespaces)

	prompter := kubecollect.NewPrompter(os.Stdin, os.Stderr)
	selected, err := prompter.SelectNamespaces(namespaces, kubeContext.Namespace)
	if err != nil {
		return err
	}

	groups, err := confirmGroups(prompter, opts)
	if err != nil {
		return err
	}
	if !groups.Any() {
		return errors.New("nothing was selected to collect")
	}

	collector := kubecollect.Collector{
		Clients:  clients,
		Context:  kubeContext,
		Options:  opts,
		Sink:     kubecollect.DirSink{Dir: output},
		Progress: os.Stderr,
	}
	result, err := collector.Run(ctx, selected, groups)
	if err != nil {
		return err
	}
	if result.Collected == 0 {
		return fmt.Errorf("nothing could be collected, see %s/collection-errors.txt", output)
	}
	if uploader == nil {
		fmt.Fprintf(os.Stderr, "\nCollected %d files into %s.\n", result.Collected, output)
		if result.Problems > 0 {
			fmt.Fprintf(os.Stderr, "%d parts could not be collected, see collection-errors.txt.\n", result.Problems)
		}
		return nil
	}
	return confirmAndUpload(ctx, prompter, *uploader, output, result)
}

func confirmAndUpload(
	ctx context.Context,
	prompter *kubecollect.Prompter,
	uploader kubecollect.Uploader,
	dir string,
	result kubecollect.Result,
) error {
	files, err := kubecollect.ListBundleFiles(dir)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "\nCollected:")
	printBundleParts(files)
	if result.Problems > 0 {
		fmt.Fprintf(os.Stderr, "%d parts could not be collected, see collection-errors.txt.\n", result.Problems)
	}
	fmt.Fprintf(os.Stderr, "\nThe files are in %s.\n", dir)
	fmt.Fprintln(os.Stderr, "You can look through them now, and edit or delete any you do not want to upload.")
	if upload, err := prompter.Confirm("Upload the files that are left to Distr?"); err != nil {
		return err
	} else if !upload {
		fmt.Fprintln(os.Stderr, "Nothing was uploaded.")
		return nil
	}

	if files, err = kubecollect.ListBundleFiles(dir); err != nil {
		return err
	} else if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "Nothing was uploaded, since no files are left.")
		return nil
	}
	uploaded, err := uploader.Upload(ctx, files, func(done int) {
		fmt.Fprintf(os.Stderr, "\rUploading %d of %d files", done, len(files))
	})
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}
	for _, failure := range uploaded.Failed {
		fmt.Fprintf(os.Stderr, "  Warning: failed to upload %s\n", failure)
	}
	fmt.Fprintf(os.Stderr, "Uploaded %d of %d files.\n", uploaded.Uploaded, len(files))
	return nil
}

// printBundleParts sums the files up per namespace, the cluster information and each file at the
// top of the bundle.
func printBundleParts(files []kubecollect.BundleFile) {
	type part struct {
		name  string
		files int
		bytes int64
	}
	var parts []*part
	byName := make(map[string]*part)
	for _, file := range files {
		segments := strings.SplitN(file.Name, "/", 3)
		name := segments[0]
		if name == "namespaces" && len(segments) == 3 {
			name = "namespace " + segments[1]
		}
		if byName[name] == nil {
			byName[name] = &part{name: name}
			parts = append(parts, byName[name])
		}
		byName[name].files++
		byName[name].bytes += file.Size
	}
	width := 0
	for _, p := range parts {
		width = max(width, len(p.name))
	}
	for _, p := range parts {
		fmt.Fprintf(os.Stderr, "  %-*s  %5d %-5s  %s\n", width, p.name, p.files, filesWord(p.files), formatBytes(p.bytes))
	}
}

func filesWord(count int) string {
	if count == 1 {
		return "file"
	}
	return "files"
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	value, exponent := float64(n)/unit, 0
	for value >= unit && exponent < 2 {
		value /= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", value, "KMG"[exponent])
}

func confirmGroups(prompter *kubecollect.Prompter, opts kubecollect.Options) (kubecollect.Groups, error) {
	included := func(include bool) string {
		if include {
			return "included"
		}
		return "not included"
	}
	var groups kubecollect.Groups
	questions := []struct {
		target   *bool
		question string
		onlyIf   *bool
	}{
		{
			target: &groups.Cluster,
			question: "Collect the cluster information (Kubernetes version, nodes, namespaces, custom resource " +
				"definitions, storage and ingress classes, admission webhooks, persistent volumes of the selected " +
				"namespaces)?",
		},
		{target: &groups.Resources, question: fmt.Sprintf(
			"Collect all resources of the selected namespaces? ConfigMap values are %s, Secret values are %s.",
			included(opts.IncludeConfigMapData), included(opts.IncludeSecretData),
		)},
		{
			target:   &groups.HelmReleases,
			question: "Collect the Helm releases of the selected namespaces (chart and app version, status, history)?",
		},
		{
			target: &groups.HelmValues,
			question: "Collect the values the Helm releases were installed with? They can contain credentials, " +
				"and Distr masks only those it recognizes.",
			onlyIf: &groups.HelmReleases,
		},
		{target: &groups.Events, question: "Collect the events of the selected namespaces?"},
		{target: &groups.Logs, question: fmt.Sprintf(
			"Collect the last %d lines of every container log, including the previous run of restarted containers?",
			opts.LogTailLines,
		)},
	}

	_, _ = fmt.Fprintln(os.Stderr)
	for _, q := range questions {
		if q.onlyIf != nil && !*q.onlyIf {
			continue
		} else if answer, err := prompter.Confirm(q.question); err != nil {
			return groups, err
		} else {
			*q.target = answer
		}
	}
	_, _ = fmt.Fprintln(os.Stderr)
	return groups, nil
}
