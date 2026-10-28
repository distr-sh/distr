package kubecollect

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

type uploadedResource struct {
	Request     string
	ContentType string
	Content     string
}

func TestUploader(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()
	sink := DirSink{Dir: dir}
	for name, content := range map[string]string{
		"summary.yaml":                        "collectorVersion: snapshot\n",
		"collection-errors.txt":               "list the secrets: forbidden\n",
		"cluster/version.json":                "{}",
		"namespaces/shop/pods/web.yaml":       "kind: Pod\n",
		"namespaces/shop/logs/web/nginx.log":  "started\n",
		"namespaces/shop/configmaps/app.yaml": "kind: ConfigMap\n",
		"namespaces/shop/events.yaml":         "broken",
	} {
		g.Expect(sink.Write(t.Context(), name, "", []byte(content))).To(Succeed())
	}
	g.Expect(os.Remove(filepath.Join(dir, "namespaces", "shop", "configmaps", "app.yaml"))).To(Succeed())

	var (
		mu       sync.Mutex
		uploaded = make(map[string]uploadedResource)
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var content []byte
		if file, _, err := r.FormFile("content"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else if content, err = io.ReadAll(file); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else if string(content) == "broken" {
			http.Error(w, "storage unavailable", http.StatusInternalServerError)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		uploaded[r.FormValue("name")] = uploadedResource{
			Request:     r.Method + " " + r.URL.RequestURI(),
			ContentType: r.FormValue("contentType"),
			Content:     string(content),
		}
	}))
	defer server.Close()

	files, err := ListBundleFiles(dir)
	g.Expect(err).NotTo(HaveOccurred())
	uploader := Uploader{URL: server.URL + "/support-bundle-collect/bundle", Secret: "s3cret", Client: server.Client()}
	result, err := uploader.Upload(t.Context(), files, nil)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Uploaded).To(Equal(5))
	g.Expect(result.Failed).To(ConsistOf(
		"namespaces/shop/events.yaml: 500 Internal Server Error: storage unavailable",
	))
	request := "POST /support-bundle-collect/bundle/resources?bundleSecret=s3cret"
	g.Expect(uploaded).To(Equal(map[string]uploadedResource{
		"summary.yaml":                       {request, "application/yaml", "collectorVersion: snapshot\n"},
		"collection-errors.txt":              {request, "text/plain", "list the secrets: forbidden\n"},
		"cluster/version.json":               {request, "application/json", "{}"},
		"namespaces/shop/pods/web.yaml":      {request, "application/yaml", "kind: Pod\n"},
		"namespaces/shop/logs/web/nginx.log": {request, "text/x-log", "started\n"},
	}))
}

func TestUploaderStopsWhenRejected(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()
	files := make([]BundleFile, 0, 50)
	for i := range 50 {
		path := filepath.Join(dir, fmt.Sprintf("%02d.yaml", i))
		g.Expect(os.WriteFile(path, []byte("kind: Pod\n"), 0o644)).To(Succeed())
		files = append(files, BundleFile{
			Name: filepath.Base(path), Path: path, ContentType: types.SupportBundleResourceContentTypeYAML,
		})
	}

	var (
		mu       sync.Mutex
		requests int
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests++
		mu.Unlock()
		http.Error(w, "support bundle is not accepting data", http.StatusBadRequest)
	}))
	defer server.Close()

	uploader := Uploader{URL: server.URL, Secret: "s3cret", Client: server.Client()}
	_, err := uploader.Upload(t.Context(), files, nil)

	rejected, ok := errors.AsType[*UploadRejectedError](err)
	g.Expect(ok).To(BeTrue())
	g.Expect(rejected.Message).To(Equal("support bundle is not accepting data"))
	mu.Lock()
	defer mu.Unlock()
	g.Expect(requests).To(BeNumerically("<=", uploadConcurrency))
}

func TestUploaderKeepsTheSecretOutOfErrors(t *testing.T) {
	g := NewWithT(t)
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	path := filepath.Join(t.TempDir(), "summary.yaml")
	g.Expect(os.WriteFile(path, nil, 0o644)).To(Succeed())

	uploader := Uploader{URL: server.URL, Secret: "s3cret", Client: http.DefaultClient}
	result, err := uploader.Upload(t.Context(), []BundleFile{{Name: "summary.yaml", Path: path}}, nil)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Failed).To(HaveLen(1))
	g.Expect(result.Failed[0]).NotTo(ContainSubstring("s3cret"))
}
