// Packs skills/distr for a release into dist/skill: assets/ holds the archives uploaded to the
// GitHub release of the tag, site/ the files synced to distr.sh that point at them.
//
// Run it from the repository root: go run ./skills <tag>
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed templates
var templatesFS embed.FS

var templates = template.Must(template.New("").Option("missingkey=error").Funcs(template.FuncMap{
	"json": func(v any) (string, error) {
		b, err := json.Marshal(v)
		return string(b), err
	},
}).ParseFS(templatesFS, "templates/*.json"))

// A release candidate is tagged before release-please bumps metadata.version, so the shipped
// SKILL.md takes its version from the tag.
var versionLine = regexp.MustCompile(`(?m)^(\s+version: )'[^']*'`)

// A fixed modification time keeps the archives from depending on when the files were checked out.
var modified = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

type file struct {
	path    string
	content []byte
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: go run ./skills <tag>")
	}
	tag := os.Args[1]

	skill := readSkill(tag)
	skillMD := skill[0].content

	// The frontmatter is the first YAML document of SKILL.md, which a decoder reads up to the next "---".
	var meta struct{ Name, Description string }
	check(yaml.NewDecoder(bytes.NewReader(skillMD)).Decode(&meta))

	archive := zipFiles(skill)
	pluginArchive := zipFiles(pluginFiles(meta.Name, skill))

	assetURL := "https://github.com/distr-sh/distr/releases/download/" + tag + "/" + meta.Name
	data := map[string]string{
		"Name":          meta.Name,
		"Description":   meta.Description,
		"Version":       tag,
		"ArchiveURL":    assetURL + ".zip",
		"ArchiveDigest": "sha256:" + sha256Hex(archive),
		"PluginURL":     assetURL + "-claude-plugin.zip",
		"PluginSHA256":  sha256Hex(pluginArchive),
	}

	site := "site/.well-known/agent-skills/"
	writeFiles("dist/skill", []file{
		{"assets/" + meta.Name + ".zip", archive},
		{"assets/" + meta.Name + "-claude-plugin.zip", pluginArchive},
		{site + "index.json", render("index.json", data)},
		{site + "marketplace.json", render("marketplace.json", data)},
		{site + meta.Name + "/SKILL.md", skillMD},
	})
}

// readSkill returns the files of skills/distr, SKILL.md first and with the tag as its version.
func readSkill(tag string) []file {
	skillFS := os.DirFS("skills/distr")
	skillMD := must(fs.ReadFile(skillFS, "SKILL.md"))
	if !versionLine.Match(skillMD) {
		log.Fatal("skills/distr/SKILL.md has no metadata.version")
	}

	files := []file{{"SKILL.md", versionLine.ReplaceAll(skillMD, []byte("${1}'"+tag+"'"))}}
	check(fs.WalkDir(skillFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && path != "SKILL.md" {
			files = append(files, file{path, must(fs.ReadFile(skillFS, path))})
		}
		return err
	}))
	return files
}

// pluginFiles lays the skill out as a Claude Code plugin.
func pluginFiles(name string, skill []file) []file {
	files := make([]file, 0, len(skill)+1)
	files = append(files, file{".claude-plugin/plugin.json", render("plugin.json", nil)})
	for _, f := range skill {
		files = append(files, file{"skills/" + name + "/" + f.path, f.content})
	}
	return files
}

func zipFiles(files []file) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range files {
		header := &zip.FileHeader{Name: f.path, Method: zip.Deflate, Modified: modified}
		header.SetMode(0o644)
		entry := must(w.CreateHeader(header))
		must(entry.Write(f.content))
	}
	check(w.Close())
	return buf.Bytes()
}

func render(name string, data any) []byte {
	var buf bytes.Buffer
	check(templates.ExecuteTemplate(&buf, name, data))
	return buf.Bytes()
}

// writeFiles replaces dir with the given files.
func writeFiles(dir string, files []file) {
	check(os.RemoveAll(dir))
	for _, f := range files {
		path := filepath.Join(dir, f.path)
		check(os.MkdirAll(filepath.Dir(path), 0o755))
		check(os.WriteFile(path, f.content, 0o644))
	}
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func must[T any](v T, err error) T {
	check(err)
	return v
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
