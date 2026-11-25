package main

import (
	"cmp"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"text/template"

	"github.com/github/go-spdx/v2/spdxexp"
	"github.com/google/go-licenses/v2/licenses"
	"github.com/spf13/pflag"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"k8s.io/klog/v2"
)

const (
	dir       = "hack/third-party-licenses"
	ownModule = "github.com/distr-sh/distr"

	componentsFile         = "components.json"
	goLicenseOverridesFile = "go-license-overrides.json"
	licenseExceptionsFile  = "license-exceptions.json"
	npmLicenseSourcesFile  = "npm-license-sources.json"

	thirdPartyLicensesOutput = "THIRD_PARTY_LICENSES.md"
	noticeOutput             = "NOTICE"
	// copied into the Distr image, too large to commit
	licenseTextsOutput = "dist/third-party-licenses.txt"
)

// Each template is named after the file it renders.
//
//go:embed templates
var templateFS embed.FS

var (
	templates = template.Must(template.New("").Option("missingkey=error").ParseFS(templateFS, "templates/*"))

	allowedLicenses = []string{
		"0BSD",
		"Apache-2.0",
		"BSD-2-Clause",
		"BSD-3-Clause",
		"ISC",
		"MIT",
		"MPL-2.0",
		"OFL-1.1",
		"PostgreSQL",
	}

	ownImages      = regexp.MustCompile(`^(ghcr\.io/distr-sh/|registry\.distr\.sh/)`)
	noticeFile     = regexp.MustCompile(`(?i)^NOTICE(\.txt|\.md)?$`)
	licenseFile    = regexp.MustCompile(`(?i)^(LICEN[CS]E|COPYING)([.-].*)?$`)
	composeImage   = regexp.MustCompile(`(?m)^\s*(?:-\s+)?image:\s*['"]?([^'"\s#]+)`)
	valuesImage    = regexp.MustCompile(`(?m)^\s*(?:-\s+)?(?:image|repository):\s*['"]?([^'"\s#]+)`)
	dockerfileFrom = regexp.MustCompile(`(?m)^FROM\s+(?:--\S+\s+)*(\S+)`)
	chartDeps      = regexp.MustCompile(`(?m)^dependencies:\n((?:[ -].*\n?)*)`)
	chartDepName   = regexp.MustCompile(`(?m)^\s*-\s+name:\s*(\S+)`)
	trailingSpace  = regexp.MustCompile(`(?m)[ \t]+$`)
	blankLines     = regexp.MustCompile(`\n{3,}`)
	imageDigest    = regexp.MustCompile(`@sha256:.*$`)
	imageTag       = regexp.MustCompile(`:[^/]+$`)

	collator = collate.New(language.English)
)

type component struct {
	Name    string   `json:"name"`
	License string   `json:"license"`
	Source  string   `json:"source"`
	Images  []string `json:"images"`
	Charts  []string `json:"charts"`
}

type licenseOverride struct {
	License string `json:"license"`
	URL     string `json:"url"`
}

type licenseException struct {
	Licenses []string `json:"licenses"`
	Reason   string   `json:"reason"`
}

type npmLicenseSource struct {
	Package string `json:"package"`
	Reason  string `json:"reason"`
}

type npmPackage struct {
	Name     string   `json:"name"`
	Versions []string `json:"versions"`
	Paths    []string `json:"paths"`
	License  string   `json:"license"`
	Homepage string   `json:"homepage"`
}

type namedText struct {
	name string
	text string
}

type textCollection struct {
	file  *regexp.Regexp
	paths map[string]bool
	texts []namedText
}

type goModule struct {
	name     string
	version  string
	licenses map[string]string
}

// sortedNames lists the names collected in a set, which are often found more than once, in the order shown in the
// generated files:
//
//	| Used for                       | Example content                                        |
//	| ------------------------------ | ------------------------------------------------------ |
//	| how a component is shipped     | Docker Compose, Helm chart, base of `Dockerfile.distr` |
//	| packages sharing a text        | github.com/aws/aws-sdk-go-v2, @sentry/core             |
//	| dependencies of the Helm chart | postgresql, rustfs, loki                               |
func sortedNames(names map[string]struct{}) []string {
	return slices.SortedFunc(maps.Keys(names), collator.CompareString)
}

type shippedImages map[string]map[string]struct{}

func (s shippedImages) add(ref, via string) {
	image := stripTag(ref)
	if ownImages.MatchString(image) {
		return
	}
	if s[image] == nil {
		s[image] = map[string]struct{}{}
	}
	s[image][via] = struct{}{}
}

type generator struct {
	errors []string
	// Apache-2.0 requires redistributions to carry the attribution notices of a dependency's NOTICE file.
	notices        *textCollection
	licenses       *textCollection
	exceptions     map[string]licenseException
	usedExceptions map[string]bool
}

func main() {
	npmReport := pflag.String("npm", "", "the output of `pnpm licenses list --prod --json`")
	pflag.Parse()
	klog.LogToStderr(false)
	klog.SetOutput(io.Discard)
	if err := run(*npmReport); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(npmReport string) error {
	if npmReport == "" {
		return errors.New("--npm is required")
	}
	npmReport, err := filepath.Abs(npmReport)
	if err != nil {
		return err
	}
	// go-licenses loads the packages of the module in the working directory
	if err := os.Chdir("../.."); err != nil {
		return err
	}

	g := &generator{
		notices:        newTextCollection(noticeFile),
		licenses:       newTextCollection(licenseFile),
		usedExceptions: map[string]bool{},
	}
	if err := readJSON(licenseExceptionsFile, &g.exceptions); err != nil {
		return err
	}

	sections := map[string]string{}
	if sections["components"], err = g.components(); err != nil {
		return err
	}
	if sections["backend"], err = g.backend(context.Background()); err != nil {
		return err
	}
	if sections["frontend"], err = g.frontend(npmReport); err != nil {
		return err
	}
	for _, name := range slices.Sorted(maps.Keys(g.exceptions)) {
		if !g.usedExceptions[name] {
			g.errorf("%s lists %s, which no longer needs an exception", licenseExceptionsFile, name)
		}
	}

	if len(g.errors) > 0 {
		for _, e := range g.errors {
			fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
		os.Exit(1)
	}

	if err := render(thirdPartyLicensesOutput, sections); err != nil {
		return err
	}
	if err := render(noticeOutput, map[string]string{"notices": g.notices.section()}); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(licenseTextsOutput), 0o755); err != nil {
		return err
	}
	return render(licenseTextsOutput, map[string]string{"licenses": g.licenses.section()})
}

func (g *generator) errorf(format string, args ...any) {
	g.errors = append(g.errors, fmt.Sprintf(format, args...))
}

func (g *generator) checkLicense(kind, name, expression string) {
	if ok, err := spdxexp.Satisfies(expression, allowedLicenses); err != nil {
		g.errorf("the license %q of %s %s is not a valid SPDX expression", expression, kind, name)
	} else if ok {
		return
	} else if exception, found := g.exceptions[name]; !found {
		g.errorf("the license %s of %s %s is not allowed", expression, kind, name)
	} else if ok, _ := spdxexp.Satisfies(expression, slices.Concat(allowedLicenses, exception.Licenses)); !ok {
		g.errorf("the license %s of %s %s is not allowed and exceeds its exception", expression, kind, name)
	} else {
		g.usedExceptions[name] = true
	}
}

func newTextCollection(file *regexp.Regexp) *textCollection {
	return &textCollection{file: file, paths: map[string]bool{}}
}

// collect adds the files in dir whose name matches and returns how many there are.
func (c *textCollection) collect(name, dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	found := 0
	for _, entry := range entries {
		if entry.Type().IsRegular() && c.file.MatchString(entry.Name()) {
			found++
			if err := c.add(name, filepath.Join(dir, entry.Name())); err != nil {
				return 0, err
			}
		}
	}
	return found, nil
}

func (c *textCollection) add(name, path string) error {
	if c.paths[path] {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	c.paths[path] = true
	c.texts = append(c.texts, namedText{name: name, text: normalizeText(string(data))})
	return nil
}

// section lists every distinct text once, under the names of all that carry it.
func (c *textCollection) section() string {
	type group struct {
		names []string
		text  string
	}
	namesByText := map[string]map[string]struct{}{}
	for _, t := range c.texts {
		if namesByText[t.text] == nil {
			namesByText[t.text] = map[string]struct{}{}
		}
		namesByText[t.text][t.name] = struct{}{}
	}
	groups := make([]group, 0, len(namesByText))
	for text, names := range namesByText {
		groups = append(groups, group{names: sortedNames(names), text: text})
	}
	slices.SortFunc(groups, func(a, b group) int {
		return cmp.Or(collator.CompareString(a.names[0], b.names[0]), strings.Compare(a.text, b.text))
	})
	var section strings.Builder
	for _, group := range groups {
		fmt.Fprintf(&section, "\n--- %s\n\n%s\n", strings.Join(group.names, ", "), group.text)
	}
	return section.String()
}

func normalizeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = trailingSpace.ReplaceAllString(text, "")
	text = blankLines.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

func (g *generator) components() (string, error) {
	images, err := findShippedImages()
	if err != nil {
		return "", err
	}
	charts, err := findShippedCharts()
	if err != nil {
		return "", err
	}
	var components []component
	if err := readJSON(componentsFile, &components); err != nil {
		return "", err
	}
	listedImages := map[string]bool{}
	listedCharts := map[string]bool{}
	var rows [][]string
	for _, c := range components {
		via := map[string]struct{}{}
		for _, image := range c.Images {
			listedImages[image] = true
			maps.Copy(via, images[image])
		}
		for _, chart := range c.Charts {
			listedCharts[chart] = true
			if _, ok := charts[chart]; ok {
				via["Helm chart"] = struct{}{}
			}
		}
		if len(via) == 0 {
			g.errorf(`%s lists "%s", which is no longer shipped`, componentsFile, c.Name)
		}
		rows = append(rows, []string{
			c.Name,
			c.License,
			strings.Join(sortedNames(via), ", "),
			link(strings.TrimPrefix(c.Source, "https://"), c.Source),
		})
	}
	for _, image := range slices.Sorted(maps.Keys(images)) {
		if !listedImages[image] {
			g.errorf("image %s is shipped but missing from %s", image, componentsFile)
		}
	}
	for _, chart := range sortedNames(charts) {
		if !listedCharts[chart] {
			g.errorf("chart %s is a dependency of the Helm chart but missing from %s", chart, componentsFile)
		}
	}
	return table([]string{"Component", "License", "Shipped as", "Source"}, rows), nil
}

func findShippedImages() (shippedImages, error) {
	found := shippedImages{}
	composeFiles, err := filepath.Glob("deploy/docker/*/docker-compose.yaml")
	if err != nil {
		return nil, err
	}
	for _, file := range composeFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		for _, match := range composeImage.FindAllStringSubmatch(string(data), -1) {
			found.add(match[1], "Docker Compose")
		}
	}
	values, err := os.ReadFile("deploy/charts/distr/values.yaml")
	if err != nil {
		return nil, err
	}
	for _, match := range valuesImage.FindAllStringSubmatch(string(values), -1) {
		found.add(match[1], "Helm chart")
	}
	var dockerfiles []string
	for _, pattern := range []string{"Dockerfile.*", "*/Dockerfile"} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		dockerfiles = append(dockerfiles, matches...)
	}
	slices.Sort(dockerfiles)
	for _, file := range dockerfiles {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		stages := dockerfileFrom.FindAllStringSubmatch(string(data), -1)
		if len(stages) == 0 {
			return nil, fmt.Errorf("%s has no FROM instruction", file)
		}
		found.add(stages[len(stages)-1][1], fmt.Sprintf("base of `%s`", file))
	}
	return found, nil
}

func findShippedCharts() (map[string]struct{}, error) {
	chart, err := os.ReadFile("deploy/charts/distr/Chart.yaml")
	if err != nil {
		return nil, err
	}
	charts := map[string]struct{}{}
	if deps := chartDeps.FindStringSubmatch(string(chart)); deps != nil {
		for _, match := range chartDepName.FindAllStringSubmatch(deps[1], -1) {
			charts[match[1]] = struct{}{}
		}
	}
	return charts, nil
}

func (g *generator) backend(ctx context.Context) (string, error) {
	var overrides map[string]licenseOverride
	if err := readJSON(goLicenseOverridesFile, &overrides); err != nil {
		return "", err
	}
	usedOverrides := map[string]bool{}

	if err := os.Setenv("GOOS", "linux"); err != nil {
		return "", err
	}
	if err := os.Setenv("GOARCH", "amd64"); err != nil {
		return "", err
	}
	classifier, err := licenses.NewClassifier()
	if err != nil {
		return "", err
	}
	libs, err := licenses.Libraries(ctx, classifier, false, []string{ownModule}, "./cmd/...")
	if err != nil {
		return "", err
	}

	modulesByKey := map[string]*goModule{}
	var modules []*goModule
	for _, lib := range libs {
		name := lib.Name()
		version := lib.Version()
		if version == "" {
			version = "Unknown"
		}
		found := map[string]string{}
		if len(lib.Licenses) == 0 {
			key := overrideFor(overrides, name)
			if key == "" {
				g.errorf("go-licenses cannot identify the license of %s, add it to %s", name, goLicenseOverridesFile)
				continue
			}
			if !usedOverrides[key] {
				usedOverrides[key] = true
				moduleDir, err := goCommand("list", "-m", "-f", "{{.Dir}}", key)
				if err != nil {
					return "", err
				}
				if _, err := g.notices.collect(key, moduleDir); err != nil {
					return "", err
				}
				if found, err := g.licenses.collect(key, moduleDir); err != nil {
					return "", err
				} else if found == 0 {
					g.errorf("Go module %s contains no license file", key)
				}
			}
			found[overrides[key].License] = overrides[key].URL
			name = key
		} else {
			if _, err := g.notices.collect(name, filepath.Dir(lib.LicenseFile)); err != nil {
				return "", err
			}
			if err := g.licenses.add(name, lib.LicenseFile); err != nil {
				return "", err
			}
			for _, license := range lib.Licenses {
				found[license.Name] = pkgGoDevURL(name, version)
			}
		}
		// go-licenses reports a library once per license file, and all of them apply.
		key := name + "@" + version
		module := modulesByKey[key]
		if module == nil {
			module = &goModule{name: name, version: version, licenses: map[string]string{}}
			modulesByKey[key] = module
			modules = append(modules, module)
		}
		maps.Copy(module.licenses, found)
	}
	for _, key := range slices.Sorted(maps.Keys(overrides)) {
		if !usedOverrides[key] {
			g.errorf("%s lists %s, which go-licenses no longer needs", goLicenseOverridesFile, key)
		}
	}

	if err := g.licenses.add("Go standard library", filepath.Join(build.Default.GOROOT, "LICENSE")); err != nil {
		return "", err
	}
	rows := [][]string{{
		"Go standard library",
		strings.TrimPrefix(runtime.Version(), "go"),
		link("BSD-3-Clause", "https://go.dev/LICENSE"),
	}}
	slices.SortStableFunc(modules, func(a, b *goModule) int { return collator.CompareString(a.name, b.name) })
	for _, module := range modules {
		names := slices.SortedFunc(maps.Keys(module.licenses), collator.CompareString)
		expression := strings.Join(names, " AND ")
		g.checkLicense("Go module", module.name, expression)
		urls := slices.Compact(slices.Sorted(maps.Values(module.licenses)))
		if len(urls) == 1 {
			expression = link(expression, urls[0])
		} else {
			linked := make([]string, len(names))
			for i, name := range names {
				linked[i] = link(name, module.licenses[name])
			}
			expression = strings.Join(linked, " AND ")
		}
		rows = append(rows, []string{module.name, module.version, expression})
	}
	return table([]string{"Module", "Version", "License"}, rows), nil
}

func overrideFor(overrides map[string]licenseOverride, name string) string {
	for prefix := range overrides {
		if name == prefix || strings.HasPrefix(name, prefix+"/") {
			return prefix
		}
	}
	return ""
}

func pkgGoDevURL(name, version string) string {
	if version == "Unknown" {
		return ""
	}
	return fmt.Sprintf("https://pkg.go.dev/%s@%s?tab=licenses", name, version)
}

func (g *generator) frontend(npmReport string) (string, error) {
	data, err := os.ReadFile(npmReport)
	if err != nil {
		return "", err
	}
	var report map[string][]npmPackage
	if err := json.Unmarshal(data, &report); err != nil {
		return "", fmt.Errorf("failed to parse %s: %w", npmReport, err)
	}
	var packages []npmPackage
	for _, group := range report {
		for _, pkg := range group {
			if pkg.Name != "@distr-sh/distr-sdk" {
				packages = append(packages, pkg)
			}
		}
	}
	slices.SortFunc(packages, func(a, b npmPackage) int { return collator.CompareString(a.Name, b.Name) })
	var licenseSources map[string]npmLicenseSource
	if err := readJSON(npmLicenseSourcesFile, &licenseSources); err != nil {
		return "", err
	}
	usedLicenseSources := map[string]bool{}
	var rows [][]string
	for _, pkg := range packages {
		for _, path := range pkg.Paths {
			if _, err := g.notices.collect(pkg.Name, path); err != nil {
				return "", err
			}
		}
		if found, err := g.collectNpmLicenses(pkg.Name, pkg.Paths); err != nil {
			return "", err
		} else if found == 0 {
			usedLicenseSources[pkg.Name] = true
			if err := g.collectNpmLicenseSource(pkg.Name, packages, licenseSources); err != nil {
				return "", err
			}
		}
		if pkg.License == "Unknown" {
			g.errorf(`the license of npm package %s is unknown, run "pnpm install --frozen-lockfile" first`, pkg.Name)
		} else {
			g.checkLicense("npm package", pkg.Name, pkg.License)
		}
		rows = append(rows, []string{link(pkg.Name, pkg.Homepage), strings.Join(pkg.Versions, ", "), pkg.License})
	}
	for _, name := range slices.Sorted(maps.Keys(licenseSources)) {
		if !usedLicenseSources[name] {
			g.errorf("%s lists %s, which no longer needs it", npmLicenseSourcesFile, name)
		}
	}
	return table([]string{"Package", "Version", "License"}, rows), nil
}

func (g *generator) collectNpmLicenses(name string, paths []string) (int, error) {
	total := 0
	for _, path := range paths {
		found, err := g.licenses.collect(name, path)
		if err != nil {
			return 0, err
		}
		total += found
	}
	return total, nil
}

func (g *generator) collectNpmLicenseSource(
	name string, packages []npmPackage, sources map[string]npmLicenseSource,
) error {
	source, ok := sources[name]
	if !ok {
		g.errorf("npm package %s contains no license file, add it to %s", name, npmLicenseSourcesFile)
		return nil
	}
	i := slices.IndexFunc(packages, func(p npmPackage) bool { return p.Name == source.Package })
	if i < 0 {
		g.errorf("%s takes the license of %s from %s, which is not a dependency",
			npmLicenseSourcesFile, name, source.Package)
		return nil
	}
	if found, err := g.collectNpmLicenses(name, packages[i].Paths); err != nil {
		return err
	} else if found == 0 {
		g.errorf("%s takes the license of %s from %s, which has no license file",
			npmLicenseSourcesFile, name, source.Package)
	}
	return nil
}

func render(output string, sections map[string]string) error {
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	if err := templates.ExecuteTemplate(f, filepath.Base(output), sections); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to render %s: %w", output, err)
	}
	return f.Close()
}

func readJSON(file string, v any) error {
	data, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to parse %s: %w", file, err)
	}
	return nil
}

func goCommand(args ...string) (string, error) {
	out, err := exec.Command("go", args...).Output()
	if err != nil {
		return "", fmt.Errorf("go %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func link(text, url string) string {
	if url == "" {
		return text
	}
	return fmt.Sprintf("[%s](%s)", text, url)
}

func table(header []string, rows [][]string) string {
	separator := make([]string, len(header))
	for i := range separator {
		separator[i] = "---"
	}
	lines := make([]string, 0, len(rows)+2)
	for _, row := range slices.Concat([][]string{header, separator}, rows) {
		lines = append(lines, "| "+strings.Join(row, " | ")+" |")
	}
	return strings.Join(lines, "\n")
}

func stripTag(image string) string {
	return imageTag.ReplaceAllString(imageDigest.ReplaceAllString(image, ""), "")
}
