package util_test

import (
	"os"
	"path"
	"testing"

	"github.com/distr-sh/distr/internal/util"
	. "github.com/onsi/gomega"
)

func TestSumFileSizesWithPrefix(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	for name, size := range map[string]int{
		"abc123-json.log":      100,
		"abc123-json.log.1":    20,
		"abc123-json.log.2.gz": 3,
		"abc123.log":           7000,
		"config.v2.json":       5000,
	} {
		g.Expect(os.WriteFile(path.Join(dir, name), make([]byte, size), 0o600)).To(Succeed())
	}
	g.Expect(os.Mkdir(path.Join(dir, "abc123-json.log.d"), 0o700)).To(Succeed())

	g.Expect(util.SumFileSizesWithPrefix(path.Join(dir, "abc123-json.log"))).To(BeEquivalentTo(123))
}
