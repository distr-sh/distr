package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/github/go-spdx/v2/spdxexp"
	"github.com/google/go-licenses/v2/licenses"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"k8s.io/klog/v2"
)

const (
	dir                    = "hack/third-party-licenses"
	ownModule              = "github.com/distr-sh/distr"
	spdxLicenseListVersion = "3.29.0"
	// copied into the Distr image, too large to commit
	licenseTextsFile = "dist/third-party-licenses.txt"
)

var (
	ownImages      = regexp.MustCompile(`^(ghcr\.io/distr-sh/|registry\.distr\.sh/)`)
	noticeFile     = regexp.MustCompile(`(?i)^NOTICE(\.txt|\.md)?$`)
	licenseFile    = regexp.MustCompile(`(?i)^(LICEN[CS]E|COPYING)([.-].*)?$`)
	composeImage   = regexp.MustCompile(`(?m)^\s*(?:-\s+)?image:\s*['"]?([^'"\s#]+)`)
	valuesImage    = regexp.MustCompile(`(?m)^\s*(?:-\s+)?(?:image|repository):\s*['"]?([^'"\s#]+)`)
	dockerfileFrom = regexp.MustCompile(`(?m)^FROM\s+(?:--\S+\s+)*(\S+)`)
	chartDeps      = regexp.MustCompile(`(?m)^dependencies:\n((?:[ -].*\n?)*)`)
	chartDepName   = regexp.MustCompile(`(?m)^\s*-\s+name:\s*(\S+)`)
	placeholder    = regexp.MustCompile(`\{\{(\w+)\}\}`)
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

type shippedImages struct {
	order []string
	via   map[string][]string
}

func (s *shippedImages) add(ref, via string) {
	image := stripTag(ref)
	if ownImages.MatchString(image) {
		return
	}
	if _, ok := s.via[image]; !ok {
		s.order = append(s.order, image)
	}
	s.via[image] = appendUnique(s.via[image], via)
}

type generator struct {
	errors []string
	// Apache-2.0 requires redistributions to carry the attribution notices of a dependency's NOTICE file.
	notices        *textCollection
	licenses       *textCollection
	osiApproved    []string
	exceptions     map[string]licenseException
	usedExceptions map[string]bool
}

func main() {
	npmReport := flag.String("npm", "", "the output of `pnpm licenses list --prod --json`")
	flag.Parse()
	klog.LogToStderr(false)
	klog.SetOutput(io.Discard)
	if err := run(*npmReport); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(npmReport string) error {
	if npmReport == "" {
		return errors.New("-npm is required")
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
	if g.osiApproved, err = osiApprovedLicenses(context.Background()); err != nil {
		return fmt.Errorf("failed to load the OSI approved licenses: %w", err)
	}
	if err := readJSON("license-exceptions.json", &g.exceptions); err != nil {
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
			g.errorf("license-exceptions.json lists %s, which no longer needs an exception", name)
		}
	}

	if len(g.errors) > 0 {
		for _, e := range g.errors {
			fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
		os.Exit(1)
	}

	if err := render("template.md", "THIRD_PARTY_LICENSES.md", sections); err != nil {
		return err
	}
	if err := render("notice.txt", "NOTICE", map[string]string{"notices": g.notices.section()}); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(licenseTextsFile), 0o755); err != nil {
		return err
	}
	return render("license-texts.txt", licenseTextsFile, map[string]string{"licenses": g.licenses.section()})
}

func (g *generator) errorf(format string, args ...any) {
	g.errors = append(g.errors, fmt.Sprintf(format, args...))
}

func osiApprovedLicenses(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf(
		"https://raw.githubusercontent.com/spdx/license-list-data/v%s/json/licenses.json", spdxLicenseListVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	var list struct {
		Licenses []struct {
			LicenseID     string `json:"licenseId"`
			IsOsiApproved bool   `json:"isOsiApproved"`
		} `json:"licenses"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	var ids []string
	for _, license := range list.Licenses {
		// spdxexp.Satisfies rejects the whole list if it contains an ID that its own copy of the list lacks
		if valid, _ := spdxexp.ValidateLicenses([]string{license.LicenseID}); license.IsOsiApproved && valid {
			ids = append(ids, license.LicenseID)
		}
	}
	return ids, nil
}

func (g *generator) checkLicense(kind, name, expression string) {
	if ok, err := spdxexp.Satisfies(expression, g.osiApproved); err != nil {
		g.errorf("the license %q of %s %s is not a valid SPDX expression", expression, kind, name)
	} else if ok {
		return
	} else if exception, found := g.exceptions[name]; !found {
		g.errorf("the license %s of %s %s is not OSI approved", expression, kind, name)
	} else if ok, _ := spdxexp.Satisfies(expression, slices.Concat(g.osiApproved, exception.Licenses)); !ok {
		g.errorf("the license %s of %s %s is not OSI approved and exceeds its exception", expression, kind, name)
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
	var groups []*group
	byText := map[string]*group{}
	for _, t := range c.texts {
		if byText[t.text] == nil {
			byText[t.text] = &group{text: t.text}
			groups = append(groups, byText[t.text])
		}
		byText[t.text].names = appendUnique(byText[t.text].names, t.name)
	}
	for _, group := range groups {
		slices.SortFunc(group.names, collator.CompareString)
	}
	slices.SortStableFunc(groups, func(a, b *group) int { return collator.CompareString(a.names[0], b.names[0]) })
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
	if err := readJSON("components.json", &components); err != nil {
		return "", err
	}
	listedImages := map[string]bool{}
	listedCharts := map[string]bool{}
	var rows [][]string
	for _, c := range components {
		var via []string
		for _, image := range c.Images {
			listedImages[image] = true
			for _, v := range images.via[image] {
				via = appendUnique(via, v)
			}
		}
		for _, chart := range c.Charts {
			listedCharts[chart] = true
			if slices.Contains(charts, chart) {
				via = appendUnique(via, "Helm chart")
			}
		}
		if len(via) == 0 {
			g.errorf(`components.json lists "%s", which is no longer shipped`, c.Name)
		}
		rows = append(rows, []string{
			c.Name,
			c.License,
			strings.Join(via, ", "),
			link(strings.TrimPrefix(c.Source, "https://"), c.Source),
		})
	}
	for _, image := range images.order {
		if !listedImages[image] {
			g.errorf("image %s is shipped but missing from components.json", image)
		}
	}
	for _, chart := range charts {
		if !listedCharts[chart] {
			g.errorf("chart %s is a dependency of the Helm chart but missing from components.json", chart)
		}
	}
	return table([]string{"Component", "License", "Shipped as", "Source"}, rows), nil
}

func findShippedImages() (*shippedImages, error) {
	found := &shippedImages{via: map[string][]string{}}
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

func findShippedCharts() ([]string, error) {
	chart, err := os.ReadFile("deploy/charts/distr/Chart.yaml")
	if err != nil {
		return nil, err
	}
	var charts []string
	if deps := chartDeps.FindStringSubmatch(string(chart)); deps != nil {
		for _, match := range chartDepName.FindAllStringSubmatch(deps[1], -1) {
			charts = appendUnique(charts, match[1])
		}
	}
	return charts, nil
}

func (g *generator) backend(ctx context.Context) (string, error) {
	var overrides map[string]licenseOverride
	if err := readJSON("go-license-overrides.json", &overrides); err != nil {
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
				g.errorf("go-licenses cannot identify the license of %s, add it to go-license-overrides.json", name)
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
			g.errorf("go-license-overrides.json lists %s, which go-licenses no longer needs", key)
		}
	}

	goVersion, err := goCommand("env", "GOVERSION")
	if err != nil {
		return "", err
	}
	if goRoot, err := goCommand("env", "GOROOT"); err != nil {
		return "", err
	} else if err := g.licenses.add("Go standard library", filepath.Join(goRoot, "LICENSE")); err != nil {
		return "", err
	}
	rows := [][]string{
		{"Go standard library", strings.TrimPrefix(goVersion, "go"), link("BSD-3-Clause", "https://go.dev/LICENSE")},
	}
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
	if err := readJSON("npm-license-sources.json", &licenseSources); err != nil {
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
			g.errorf("npm-license-sources.json lists %s, which no longer needs it", name)
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
		g.errorf("npm package %s contains no license file, add it to npm-license-sources.json", name)
		return nil
	}
	i := slices.IndexFunc(packages, func(p npmPackage) bool { return p.Name == source.Package })
	if i < 0 {
		g.errorf("npm-license-sources.json takes the license of %s from %s, which is not a dependency", name, source.Package)
		return nil
	}
	if found, err := g.collectNpmLicenses(name, packages[i].Paths); err != nil {
		return err
	} else if found == 0 {
		g.errorf("npm-license-sources.json takes the license of %s from %s, which has no license file", name, source.Package)
	}
	return nil
}

func render(template, output string, sections map[string]string) error {
	data, err := os.ReadFile(filepath.Join(dir, template))
	if err != nil {
		return err
	}
	rendered := placeholder.ReplaceAllStringFunc(string(data), func(match string) string {
		return sections[placeholder.FindStringSubmatch(match)[1]]
	})
	return os.WriteFile(output, []byte(rendered), 0o644)
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

func appendUnique(list []string, value string) []string {
	if slices.Contains(list, value) {
		return list
	}
	return append(list, value)
}
