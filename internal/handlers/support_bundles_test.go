package handlers

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/distr-sh/distr/internal/dbcrypto"
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

func resource(
	name string,
	contentType types.SupportBundleResourceContentType,
	content string,
) types.SupportBundleResource {
	return types.SupportBundleResource{
		SupportBundleResourceSummary: types.SupportBundleResourceSummary{
			ID: uuid.New(), Name: name, ContentType: contentType,
		},
		Content: dbcrypto.String(content),
	}
}

func textResource(name string) types.SupportBundleResource {
	return resource(name, types.SupportBundleResourceContentTypeText, "")
}

func TestWriteSupportBundleZip_keepsEntriesInsideTheArchive(t *testing.T) {
	g := NewWithT(t)
	blank := textResource("  /  ")
	longSegment := strings.Repeat("ä", 100)
	deep := textResource(strings.Repeat(strings.Repeat("a", 128)+"/", 600) + "x")

	entries := zipEntries(g, []types.SupportBundleResource{
		textResource("../../etc/passwd"),
		textResource("/absolute/path"),
		textResource(`..\..\windows\system`),
		textResource("C:/drive"),
		textResource("a//b/./c"),
		textResource(longSegment + "/x"),
		blank,
		deep,
	})

	g.Expect(entries).To(HaveLen(8))
	g.Expect(entries).To(HaveKey("etc/passwd.txt"))
	g.Expect(entries).To(HaveKey("absolute/path.txt"))
	g.Expect(entries).To(HaveKey("windows/system.txt"))
	g.Expect(entries).To(HaveKey("C-/drive.txt"))
	g.Expect(entries).To(HaveKey("a/b/c.txt"))
	g.Expect(entries).To(HaveKey(strings.Repeat("ä", 64) + "/x.txt"))
	g.Expect(entries).To(HaveKey(blank.ID.String() + ".txt"))
	g.Expect(entries).To(HaveKey(deep.ID.String() + ".txt"))
}

func TestWriteSupportBundleZip_namesEntriesByContentType(t *testing.T) {
	g := NewWithT(t)

	entries := zipEntries(g, []types.SupportBundleResource{
		resource("cluster/nodes", types.SupportBundleResourceContentTypeYAML, "first"),
		resource("cluster/nodes.yaml", types.SupportBundleResourceContentTypeYAML, "second"),
		resource("cluster/version", types.SupportBundleResourceContentTypeJSON, ""),
		resource("pods/web/app", types.SupportBundleResourceContentTypeLog, ""),
		textResource("container-web-logs"),
		textResource("container-web-logs"),
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
