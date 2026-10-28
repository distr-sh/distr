package kubecollect

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

// Prompter asks the customer on the terminal. The bootstrap script runs the collector with its input
// read from /dev/tty, because its own input is the script piped into sh.
type Prompter struct {
	in  *bufio.Reader
	out io.Writer
}

func NewPrompter(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{in: bufio.NewReader(in), out: out}
}

func (p *Prompter) SelectNamespaces(options []NamespaceOption, defaultNamespace string) ([]string, error) {
	_, _ = fmt.Fprintln(p.out, "Namespaces:")
	for i, option := range options {
		if len(option.HelmReleases) > 0 {
			_, _ = fmt.Fprintf(p.out, "  [%d] %s (%s)\n", i+1, option.Name, strings.Join(option.HelmReleases, ", "))
		} else {
			_, _ = fmt.Fprintf(p.out, "  [%d] %s\n", i+1, option.Name)
		}
	}
	for {
		_, _ = fmt.Fprintf(p.out, "Namespaces to collect, as numbers or names separated by commas [%s]: ", defaultNamespace)
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
		if n, err := strconv.Atoi(token); err == nil {
			if n < 1 || n > len(options) {
				return nil, fmt.Errorf("%d is not a number from the list", n)
			}
			name = options[n-1].Name
		} else if !slices.ContainsFunc(options, func(o NamespaceOption) bool { return o.Name == token }) {
			return nil, fmt.Errorf("%q is not a namespace from the list", token)
		}
		if !slices.Contains(selected, name) {
			selected = append(selected, name)
		}
	}
	return selected, nil
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
