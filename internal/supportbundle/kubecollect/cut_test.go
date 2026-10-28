package kubecollect

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func TestKeepTail(t *testing.T) {
	g := NewWithT(t)
	var log strings.Builder
	for i := range 100 {
		log.WriteString(strings.Repeat("x", i%7))
		log.WriteString("line\n")
	}

	content, cut := keepTail([]byte(log.String()), 200)
	g.Expect(cut).To(BeTrue())
	g.Expect(len(content)).To(BeNumerically("<=", 200))
	g.Expect(string(content)).To(HavePrefix("--- distr: earlier output truncated at 200 bytes ---\n"))
	_, tail, _ := strings.Cut(string(content), "\n")
	g.Expect(log.String()).To(HaveSuffix("\n"+tail), "the tail starts at a line")

	content, cut = keepTail([]byte("short\n"), 200)
	g.Expect(cut).To(BeFalse())
	g.Expect(string(content)).To(Equal("short\n"))
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
