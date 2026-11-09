package supportcollector

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type Config struct {
	ConfigFlags *genericclioptions.ConfigFlags
	// Output is the directory the bundle is written into. It must be new or empty.
	Output string
	// WriteKubeconfig is the file the kubeconfig of the selected context is written to, if set.
	WriteKubeconfig string
	Options         Options
	// Uploader uploads the files the customer keeps. Without one, the bundle stays in Output.
	Uploader *Uploader
}

// Run asks the customer what to collect, collects it into the output directory and, once the
// customer confirms, uploads the files that are left.
func Run(ctx context.Context, config Config, in io.Reader, out io.Writer) error {
	if entries, err := os.ReadDir(config.Output); err == nil && len(entries) > 0 {
		return fmt.Errorf("%s is not empty, write the bundle into a new or empty directory", config.Output)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	prompter := NewPrompter(in, out)
	if err := selectKubeContext(prompter, config.ConfigFlags); err != nil {
		return err
	}
	kubeContext, err := LoadKubeContext(config.ConfigFlags)
	if err != nil {
		return err
	}
	if config.WriteKubeconfig != "" {
		if err := WriteKubeconfig(config.ConfigFlags, kubeContext, config.WriteKubeconfig); err != nil {
			return err
		}
	}
	_, _ = fmt.Fprintf(out, "Context:   %s\nCluster:   %s\nNamespace: %s\n\n",
		kubeContext.Name, kubeContext.Server, kubeContext.Namespace)

	clients, err := NewClients(kubeContext.Config)
	if err != nil {
		return err
	}

	namespaces, err := ListNamespaces(ctx, clients)
	if err != nil {
		_, _ = fmt.Fprintf(out, "The namespaces of the cluster cannot be listed (%v),\n", err)
		_, _ = fmt.Fprintf(out, "so only the namespace of the context is offered.\n\n")
		namespaces = []NamespaceOption{{Name: kubeContext.Namespace}}
	}
	AddHelmReleases(ctx, clients, namespaces)

	selected, err := prompter.SelectNamespaces(namespaces, kubeContext.Namespace)
	if err != nil {
		return err
	}

	groups, err := prompter.ConfirmGroups(config.Options)
	if err != nil {
		return err
	}
	if !groups.Any() {
		return errors.New("nothing was selected to collect")
	}

	collector := Collector{
		Clients:  clients,
		Context:  kubeContext,
		Options:  config.Options,
		Sink:     DirSink{Dir: config.Output},
		Progress: out,
	}
	result, err := collector.Run(ctx, selected, groups)
	if err != nil {
		return err
	}
	if result.Collected == 0 && result.Problems > 0 {
		return fmt.Errorf("nothing could be collected, see %s/collection-errors.txt", config.Output)
	}
	if config.Uploader == nil {
		_, _ = fmt.Fprintf(out, "\nCollected %d files into %s.\n", result.Collected, config.Output)
		if result.Problems > 0 {
			_, _ = fmt.Fprintf(out, "%d parts could not be collected, see collection-errors.txt.\n", result.Problems)
		}
		return nil
	}
	return confirmAndUpload(ctx, prompter, *config.Uploader, config.Output, result)
}

// selectKubeContext asks for the context unless --context names one or the kubeconfig has only one.
func selectKubeContext(prompter *Prompter, configFlags *genericclioptions.ConfigFlags) error {
	if configFlags.Context != nil && *configFlags.Context != "" {
		return nil
	}
	options, current, err := ListKubeContexts(configFlags)
	if err != nil {
		return err
	}
	var name string
	switch len(options) {
	case 0:
		return nil
	case 1:
		name = options[0].Name
	default:
		if name, err = prompter.SelectKubeContext(options, current); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(prompter.out)
	}
	configFlags.Context = &name
	return nil
}

func confirmAndUpload(ctx context.Context, prompter *Prompter, uploader Uploader, dir string, result Result) error {
	out := prompter.out
	files, err := ListBundleFiles(dir)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(out, "\nCollected:")
	printBundleParts(out, files)
	if result.Problems > 0 {
		_, _ = fmt.Fprintf(out, "%d parts could not be collected, see collection-errors.txt.\n", result.Problems)
	}
	_, _ = fmt.Fprintf(out, "\nThe files are in %s.\n", dir)
	_, _ = fmt.Fprintln(out, "You can look through them now, and edit or delete any you do not want to upload.")
	if upload, err := prompter.Confirm("Upload the files that are left to Distr?"); err != nil {
		return err
	} else if !upload {
		_, _ = fmt.Fprintln(out, "Nothing was uploaded.")
		return nil
	}

	if files, err = ListBundleFiles(dir); err != nil {
		return err
	} else if len(files) == 0 {
		_, _ = fmt.Fprintln(out, "Nothing was uploaded, since no files are left.")
		return nil
	}
	uploaded, err := uploader.Upload(ctx, files, func(done int) {
		_, _ = fmt.Fprintf(out, "\rUploading %d of %d files", done, len(files))
	})
	_, _ = fmt.Fprintln(out)
	if err != nil {
		return err
	}
	for _, failure := range uploaded.Failed {
		_, _ = fmt.Fprintf(out, "  Warning: failed to upload %s\n", failure)
	}
	_, _ = fmt.Fprintf(out, "Uploaded %d of %d files.\n", uploaded.Uploaded, len(files))
	return nil
}

// printBundleParts sums the files up per namespace, the cluster information and each file at the
// top of the bundle.
func printBundleParts(out io.Writer, files []BundleFile) {
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
		_, _ = fmt.Fprintf(out, "  %-*s  %5d %-5s  %s\n", width, p.name, p.files, filesWord(p.files), formatBytes(p.bytes))
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
