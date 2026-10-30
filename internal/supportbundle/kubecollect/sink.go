package kubecollect

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Sink receives every file of a bundle. The name is a slash-separated path including the file
// extension, which is also the name of the resource in Distr.
type Sink interface {
	Write(ctx context.Context, name string, contentType string, content []byte) error
}

// DirSink writes the bundle into a local directory.
type DirSink struct {
	Dir string
}

func (s DirSink) Write(_ context.Context, name string, _ string, content []byte) error {
	local := filepath.FromSlash(name)
	if !filepath.IsLocal(local) {
		return fmt.Errorf("refusing to write %q outside of %s", name, s.Dir)
	}
	path := filepath.Join(s.Dir, local)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o600)
}
