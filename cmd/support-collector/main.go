package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/distr-sh/distr/internal/supportcollector"
	"github.com/spf13/cobra"
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

	if err := newCommand().Execute(); err != nil {
		os.Exit(1)
	}
}

func newCommand() *cobra.Command {
	config := supportcollector.Config{
		ConfigFlags: supportcollector.NewConfigFlags(),
		Options:     supportcollector.DefaultOptions(),
	}
	opts := &config.Options
	cmd := &cobra.Command{
		Use:   "support-collector",
		Short: "Collect a Distr support bundle from a Kubernetes cluster",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if config.Output == "" {
				return errors.New("--output is required")
			} else if opts.LogTailLines < 0 {
				return errors.New("--log-tail-lines must not be negative")
			} else if opts.MaxResourceBytes < supportcollector.MinResourceBytes {
				return fmt.Errorf("--max-resource-bytes must be at least %d", supportcollector.MinResourceBytes)
			} else if uploader, err := uploaderFromEnv(); err != nil {
				return err
			} else {
				config.Uploader = uploader
				return nil
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			// Ctrl+C keeps the default handling and ends the process: the prompts block on the terminal
			// without a context, and the collector records a canceled request as a problem and continues.
			// The collect script owns the output directory and removes it when it is interrupted.
			return supportcollector.Run(cmd.Context(), config, os.Stdin, os.Stderr)
		},
	}

	flags := cmd.Flags()
	config.ConfigFlags.AddFlags(flags)
	flags.StringVar(&config.Output, "output", "", "directory to write the bundle into, and to upload it from when "+
		uploadURLEnv+" and "+uploadSecretEnv+" are set")
	flags.StringVar(&config.WriteKubeconfig, "write-kubeconfig", "",
		"file to write a kubeconfig with only the selected context and namespace to, for tools run afterwards")
	flags.Int64Var(&opts.LogTailLines, "log-tail-lines", opts.LogTailLines,
		"number of lines collected from the end of every container log")
	flags.IntVar(&opts.MaxResourceBytes, "max-resource-bytes", opts.MaxResourceBytes,
		"maximum size of every file of the bundle")
	flags.BoolVar(&opts.IncludeConfigMapData, "include-configmap-data", opts.IncludeConfigMapData,
		"collect the values of ConfigMaps")
	flags.BoolVar(&opts.IncludeSecretData, "include-secret-data", opts.IncludeSecretData,
		"collect the decoded values of Secrets, except service account and bootstrap tokens and Helm releases")
	return cmd
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
