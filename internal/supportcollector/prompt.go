package supportcollector

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

var ErrNoAnswer = errors.New("the input ended without an answer")

// Prompter asks the customer on the terminal. The collect script runs the support collector with its
// input read from /dev/tty, because its own input is the script piped into sh.
type Prompter struct {
	in  *bufio.Reader
	out io.Writer
}

func NewPrompter(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{in: bufio.NewReader(in), out: out}
}

// SelectKubeContext offers the current context as the default, or no default when the kubeconfig
// has none or names one it does not contain.
func (p *Prompter) SelectKubeContext(options []KubeContextOption, current string) (string, error) {
	_, _ = fmt.Fprintln(p.out, "Contexts:")
	for i, option := range options {
		if option.Server != "" {
			_, _ = fmt.Fprintf(p.out, "  [%d] %s (%s)\n", i+1, option.Name, option.Server)
		} else {
			_, _ = fmt.Fprintf(p.out, "  [%d] %s\n", i+1, option.Name)
		}
	}
	if !slices.ContainsFunc(options, func(o KubeContextOption) bool { return o.Name == current }) {
		current = ""
	}
	for {
		if current != "" {
			_, _ = fmt.Fprintf(p.out, "Context to collect from, as a number or name, or press Enter for %q: ", current)
		} else {
			_, _ = fmt.Fprint(p.out, "Context to collect from, as a number or name: ")
		}
		answer, err := p.readLine()
		if err != nil {
			return "", err
		}
		if selected, err := parseKubeContextSelection(answer, options, current); err != nil {
			_, _ = fmt.Fprintln(p.out, err)
		} else {
			return selected, nil
		}
	}
}

func parseKubeContextSelection(answer string, options []KubeContextOption, current string) (string, error) {
	if answer == "" {
		if current == "" {
			return "", errors.New("select a context from the list")
		}
		return current, nil
	}
	if slices.ContainsFunc(options, func(o KubeContextOption) bool { return o.Name == answer }) {
		return answer, nil
	} else if n, err := strconv.Atoi(answer); err != nil {
		return "", fmt.Errorf("%q is not a context from the list", answer)
	} else if n < 1 || n > len(options) {
		return "", fmt.Errorf("%d is not a number from the list", n)
	} else {
		return options[n-1].Name, nil
	}
}

func (p *Prompter) SelectNamespaces(options []NamespaceOption, defaultNamespace string) ([]string, error) {
	_, _ = fmt.Fprintln(p.out, "Namespaces:")
	for i, option := range options {
		if len(option.HelmReleases) > 0 {
			_, _ = fmt.Fprintf(p.out, "  [%d] %s (Helm releases: %s)\n",
				i+1, option.Name, strings.Join(option.HelmReleases, ", "))
		} else {
			_, _ = fmt.Fprintf(p.out, "  [%d] %s\n", i+1, option.Name)
		}
	}
	for {
		_, _ = fmt.Fprintf(p.out, "Namespaces to collect, as numbers or names separated by commas, or press Enter for %q: ",
			defaultNamespace)
		answer, err := p.readLine()
		if err != nil {
			return nil, err
		}
		if selected, err := parseNamespaceSelection(answer, options, defaultNamespace); err != nil {
			_, _ = fmt.Fprintln(p.out, err)
		} else {
			return selected, nil
		}
	}
}

func parseNamespaceSelection(answer string, options []NamespaceOption, defaultNamespace string) ([]string, error) {
	tokens := strings.FieldsFunc(answer, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
	if len(tokens) == 0 {
		return []string{defaultNamespace}, nil
	}
	var selected []string
	for _, token := range tokens {
		name := token
		if !slices.ContainsFunc(options, func(o NamespaceOption) bool { return o.Name == token }) {
			if n, err := strconv.Atoi(token); err != nil {
				return nil, fmt.Errorf("%q is not a namespace from the list", token)
			} else if n < 1 || n > len(options) {
				return nil, fmt.Errorf("%d is not a number from the list", n)
			} else {
				name = options[n-1].Name
			}
		}
		if !slices.Contains(selected, name) {
			selected = append(selected, name)
		}
	}
	return selected, nil
}

func (p *Prompter) ConfirmGroups(opts Options) (Groups, error) {
	included := func(include bool) string {
		if include {
			return "included"
		}
		return "not included"
	}
	var groups Groups
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

	_, _ = fmt.Fprintln(p.out)
	for _, q := range questions {
		if q.onlyIf != nil && !*q.onlyIf {
			continue
		} else if answer, err := p.Confirm(q.question); err != nil {
			return groups, err
		} else {
			*q.target = answer
		}
	}
	_, _ = fmt.Fprintln(p.out)
	return groups, nil
}

func (p *Prompter) Confirm(question string) (bool, error) {
	for {
		_, _ = fmt.Fprintf(p.out, "%s [Y/n]: ", question)
		answer, err := p.readLine()
		if err != nil {
			return false, err
		}
		switch strings.ToLower(answer) {
		case "", "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
	}
}

func (p *Prompter) readLine() (string, error) {
	line, err := p.in.ReadString('\n')
	if errors.Is(err, io.EOF) && line == "" {
		return "", ErrNoAnswer
	} else if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
