package kubecollect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func TestDirSink(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()
	sink := DirSink{Dir: filepath.Join(dir, "bundle")}

	g.Expect(sink.Write(t.Context(), "namespaces/shop/pods/web/nginx.log", types.SupportBundleResourceContentTypeLog,
		[]byte("started\n"))).To(Succeed())
	g.Expect(os.ReadFile(filepath.Join(dir, "bundle", "namespaces", "shop", "pods", "web", "nginx.log"))).
		To(Equal([]byte("started\n")))

	for _, name := range []string{"../escaped.yaml", "/etc/escaped.yaml"} {
		g.Expect(sink.Write(t.Context(), name, types.SupportBundleResourceContentTypeYAML, nil)).NotTo(Succeed())
	}
	g.Expect(filepath.Join(dir, "escaped.yaml")).NotTo(BeAnExistingFile())
}
