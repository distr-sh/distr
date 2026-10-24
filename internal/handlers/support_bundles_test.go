package handlers

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func zipEntries(g Gomega, resources []types.SupportBundleResource) map[string]string {
	var buffer bytes.Buffer
	g.Expect(writeSupportBundleZip(&buffer, resources)).To(Succeed())
	reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	g.Expect(err).NotTo(HaveOccurred())
	entries := make(map[string]string, len(reader.File))
	for _, file := range reader.File {
		f, err := file.Open()
		g.Expect(err).NotTo(HaveOccurred())
		content, err := io.ReadAll(f)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(f.Close()).To(Succeed())
		entries[file.Name] = string(content)
	}
	return entries
}

func textResource(name string) types.SupportBundleResource {
	return types.SupportBundleResource{ID: uuid.New(), Name: name, ContentType: types.SupportBundleResourceContentTypeText}
}

func TestWriteSupportBundleZip_keepsEntriesInsideTheArchive(t *testing.T) {
	g := NewWithT(t)
	blank := textResource("  /  ")
	longSegment := strings.Repeat("ä", 100)

	entries := zipEntries(g, []types.SupportBundleResource{
		textResource("../../etc/passwd"),
		textResource("/absolute/path"),
		textResource(`..\..\windows\system`),
		textResource("C:/drive"),
		textResource("a//b/./c"),
		textResource(longSegment + "/x"),
		blank,
	})

	g.Expect(entries).To(HaveLen(7))
	g.Expect(entries).To(HaveKey("etc/passwd.txt"))
	g.Expect(entries).To(HaveKey("absolute/path.txt"))
	g.Expect(entries).To(HaveKey("windows/system.txt"))
	g.Expect(entries).To(HaveKey("C-/drive.txt"))
	g.Expect(entries).To(HaveKey("a/b/c.txt"))
	g.Expect(entries).To(HaveKey(strings.Repeat("ä", 64) + "/x.txt"))
	g.Expect(entries).To(HaveKey(blank.ID.String() + ".txt"))
}

func TestWriteSupportBundleZip_namesEntriesByContentType(t *testing.T) {
	g := NewWithT(t)

	entries := zipEntries(g, []types.SupportBundleResource{
		{Name: "cluster/nodes", ContentType: types.SupportBundleResourceContentTypeYAML, Content: "first"},
		{Name: "cluster/nodes.yaml", ContentType: types.SupportBundleResourceContentTypeYAML, Content: "second"},
		{Name: "cluster/version", ContentType: types.SupportBundleResourceContentTypeJSON},
		{Name: "pods/web/app", ContentType: types.SupportBundleResourceContentTypeLog},
		{Name: "container-web-logs", ContentType: types.SupportBundleResourceContentTypeText},
		{Name: "container-web-logs", ContentType: types.SupportBundleResourceContentTypeText},
	})

	g.Expect(entries).To(Equal(map[string]string{
		"cluster/nodes.yaml":       "first",
		"cluster/nodes-2.yaml":     "second",
		"cluster/version.json":     "",
		"pods/web/app.log":         "",
		"container-web-logs.txt":   "",
		"container-web-logs-2.txt": "",
	}))
}
