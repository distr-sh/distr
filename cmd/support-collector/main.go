package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/distr-sh/distr/internal/supportcollector"
	"github.com/spf13/pflag"
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

	config := supportcollector.Config{
		ConfigFlags: supportcollector.NewConfigFlags(),
		Options:     supportcollector.DefaultOptions(),
	}
	opts := &config.Options
	config.ConfigFlags.AddFlags(pflag.CommandLine)
	pflag.StringVar(&config.Output, "output", "", "directory to write the bundle into, and to upload it from when "+
		uploadURLEnv+" and "+uploadSecretEnv+" are set")
	pflag.StringVar(&config.WriteKubeconfig, "write-kubeconfig", "",
		"file to write a kubeconfig with only the selected context and namespace to, for tools run afterwards")
	pflag.Int64Var(&opts.LogTailLines, "log-tail-lines", opts.LogTailLines,
		"number of lines collected from the end of every container log")
	pflag.IntVar(&opts.MaxResourceBytes, "max-resource-bytes", opts.MaxResourceBytes,
		"maximum size of every file of the bundle")
	pflag.BoolVar(&opts.IncludeConfigMapData, "include-configmap-data", opts.IncludeConfigMapData,
		"collect the values of ConfigMaps")
	pflag.BoolVar(&opts.IncludeSecretData, "include-secret-data", opts.IncludeSecretData,
		"collect the decoded values of Secrets, except service account and bootstrap tokens and Helm releases")
	pflag.Parse()

	if config.Output == "" {
		usageError("--output is required")
	} else if pflag.NArg() > 0 {
		usageError(fmt.Sprintf("unexpected arguments: %v", pflag.Args()))
	} else if opts.LogTailLines < 0 {
		usageError("--log-tail-lines must not be negative")
	} else if opts.MaxResourceBytes < supportcollector.MinResourceBytes {
		usageError(fmt.Sprintf("--max-resource-bytes must be at least %d", supportcollector.MinResourceBytes))
	}
	if uploader, err := uploaderFromEnv(); err != nil {
		usageError(err.Error())
	} else {
		config.Uploader = uploader
	}

	// Ctrl+C keeps the default handling and ends the process: the prompts block on the terminal
	// without a context, and the collector records a canceled request as a problem and continues.
	// The collect script owns the output directory and removes it when it is interrupted.
	if err := supportcollector.Run(context.Background(), config, os.Stdin, os.Stderr); err != nil {
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
func uploaderFromEnv() (*supportcollector.Uploader, error) {
	uploadURL, secret := os.Getenv(uploadURLEnv), os.Getenv(uploadSecretEnv)
	if uploadURL == "" && secret == "" {
		return nil, nil
	} else if uploadURL == "" || secret == "" {
		return nil, fmt.Errorf("%s and %s must be set together", uploadURLEnv, uploadSecretEnv)
	}
	return &supportcollector.Uploader{URL: uploadURL, Secret: secret, Client: http.DefaultClient}, nil
}
