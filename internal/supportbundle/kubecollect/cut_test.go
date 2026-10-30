package kubecollect

import (
	"io"
	"strings"
	"testing"
	"testing/iotest"

	. "github.com/onsi/gomega"
)

func TestKeepTail(t *testing.T) {
	g := NewWithT(t)
	var log strings.Builder
	for i := range 100 {
		log.WriteString(strings.Repeat("x", i%7))
		log.WriteString("line\n")
	}

	for _, reader := range []io.Reader{
		strings.NewReader(log.String()),
		iotest.OneByteReader(strings.NewReader(log.String())),
	} {
		content, err := keepTail(reader, 200)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(len(content)).To(BeNumerically("<=", 200))
		g.Expect(string(content)).To(HavePrefix("--- distr: earlier output truncated at 200 bytes ---\n"))
		_, tail, _ := strings.Cut(string(content), "\n")
		g.Expect(log.String()).To(HaveSuffix("\n"+tail), "the tail starts at a line")
	}

	g.Expect(keepTail(strings.NewReader("short\n"), 200)).To(Equal([]byte("short\n")))
}

func TestKeepHead(t *testing.T) {
	g := NewWithT(t)
	yaml := strings.Repeat("key: value\n", 100)

	content, cut := keepHead([]byte(yaml), 200)
	g.Expect(cut).To(BeTrue())
	g.Expect(content).To(HaveLen(200))
	note, head, _ := strings.Cut(string(content), "\n")
	g.Expect(note).To(Equal("# --- distr: output truncated at 200 bytes ---"))
	g.Expect(yaml).To(HavePrefix(head))
}
