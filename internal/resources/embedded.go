package resources

import (
	"embed"
	"fmt"
	"io/fs"
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

func GetTemplate(name string) (*template.Template, error) {
	if tmpl, ok := templates.Load(name); ok {
		return tmpl.(*template.Template), nil
	}
	tmpl, err := template.ParseFS(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %v: %w", name, err)
	}
	templates.Store(name, tmpl)
	return tmpl, nil
}
