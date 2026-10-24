package resources

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"text/template"

	"github.com/distr-sh/distr/internal/util"
)

var (
	//go:embed embedded
	embeddedFs embed.FS
	fsys       = util.Require(fs.Sub(embeddedFs, "embedded"))
	templates  sync.Map
)

func Get(name string) ([]byte, error) {
	return fs.ReadFile(fsys, name)
}

// GetTemplate returns the template in the file name, which can use the templates defined in the
// files includes.
func GetTemplate(name string, includes ...string) (*template.Template, error) {
	files := append([]string{name}, includes...)
	key := strings.Join(files, "\x00")
	if tmpl, ok := templates.Load(key); ok {
		return tmpl.(*template.Template), nil
	}
	tmpl, err := template.ParseFS(fsys, files...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %v: %w", name, err)
	}
	templates.Store(key, tmpl)
	return tmpl, nil
}
