package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/distr-sh/distr/internal/supportbundle/kubecollect"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
)

func main() {
	// client-go logs through klog, for example when it retries a request, which would interleave with
	// the prompts. Everything the customer needs to know ends up in collection-errors.txt.
	klog.LogToStderr(false)
	klog.SetOutput(io.Discard)

	configFlags := genericclioptions.NewConfigFlags(true)
	configFlags.AddFlags(pflag.CommandLine)
	opts := kubecollect.DefaultOptions()
	output := pflag.String("output", "", "directory to write the bundle into")
	pflag.Int64Var(&opts.LogTailLines, "log-tail-lines", opts.LogTailLines,
		"number of lines collected from the end of every container log")
	pflag.IntVar(&opts.MaxResourceBytes, "max-resource-bytes", opts.MaxResourceBytes,
		"maximum size of every file of the bundle")
	pflag.BoolVar(&opts.IncludeConfigMapData, "include-configmap-data", opts.IncludeConfigMapData,
		"collect the values of ConfigMaps")
	pflag.BoolVar(&opts.IncludeSecretData, "include-secret-data", opts.IncludeSecretData,
		"collect the decoded values of Secrets, except service account tokens and Helm releases")
	pflag.Parse()

	if *output == "" {
		usageError("--output is required")
	} else if pflag.NArg() > 0 {
		usageError(fmt.Sprintf("unexpected arguments: %v", pflag.Args()))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, configFlags, *output, opts)
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func usageError(message string) {
	fmt.Fprintln(os.Stderr, message)
	pflag.Usage()
	os.Exit(2)
}

func run(
	ctx context.Context,
	configFlags *genericclioptions.ConfigFlags,
	output string,
	opts kubecollect.Options,
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
	fmt.Fprintf(os.Stderr, "\nCollected %d files into %s.\n", result.Collected, output)
	if result.Problems > 0 {
		fmt.Fprintf(os.Stderr, "%d parts could not be collected, see collection-errors.txt.\n", result.Problems)
	}
	return nil
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
	}{
		{
			&groups.Cluster,
			"Collect the cluster information (Kubernetes version, nodes, namespaces, custom resource definitions)?",
		},
		{&groups.Resources, fmt.Sprintf(
			"Collect all resources of the selected namespaces? ConfigMap values are %s, Secret values are %s.",
			included(opts.IncludeConfigMapData), included(opts.IncludeSecretData),
		)},
		{&groups.Events, "Collect the events of the selected namespaces?"},
		{&groups.Logs, fmt.Sprintf(
			"Collect the last %d lines of every container log, including the previous run of restarted containers?",
			opts.LogTailLines,
		)},
	}

	_, _ = fmt.Fprintln(os.Stderr)
	for _, q := range questions {
		if answer, err := prompter.Confirm(q.question); err != nil {
			return groups, err
		} else {
			*q.target = answer
		}
	}
	_, _ = fmt.Fprintln(os.Stderr)
	return groups, nil
}
