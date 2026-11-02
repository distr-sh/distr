package chartcache_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/distr-sh/distr/internal/chartcache"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var errOffline = errors.New("registry unreachable")

type puller struct {
	t     *testing.T
	pulls int
}

func (p *puller) pull(content string) func() (string, error) {
	return func() (string, error) {
		p.pulls++
		path := filepath.Join(p.t.TempDir(), "app.tgz")
		return path, os.WriteFile(path, []byte(content), 0o600)
	}
}

func (p *puller) fail() (string, error) {
	p.pulls++
	return "", errOffline
}

func read(path string, err error) (string, error) {
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	return string(data), err
}

func TestLocateWithDigest(t *testing.T) {
	g := NewWithT(t)
	contentDir := t.TempDir()
	cache := chartcache.New(contentDir)
	p := &puller{t: t}
	digestA := "sha256:" + strings.Repeat("a", 64)
	digestB := "sha256:" + strings.Repeat("b", 64)
	locate := func(
		cache *chartcache.Cache,
		deploymentID uuid.UUID,
		digest string,
		pull func() (string, error),
	) (string, error) {
		return read(cache.Locate(zap.NewNop(), deploymentID, uuid.New(), digest, pull))
	}

	deploymentID := uuid.New()
	g.Expect(locate(cache, deploymentID, digestA, p.pull("a"))).To(Equal("a"))
	g.Expect(locate(cache, deploymentID, digestA, p.fail)).
		To(Equal("a"), "a new revision with the same digest must not pull")
	g.Expect(locate(cache, uuid.New(), digestA, p.fail)).To(Equal("a"), "another deployment shares the chart")
	g.Expect(locate(chartcache.New(contentDir), deploymentID, digestA, p.fail)).
		To(Equal("a"), "the chart must outlive a controller restart")
	g.Expect(p.pulls).To(Equal(1))

	_, err := locate(cache, deploymentID, digestB, p.fail)
	g.Expect(err).To(MatchError(errOffline), "the cached chart must not stand in for another digest")
	g.Expect(locate(cache, deploymentID, digestB, p.pull("b"))).To(Equal("b"))
}

func TestLocateWithoutDigest(t *testing.T) {
	g := NewWithT(t)
	cache := chartcache.New(t.TempDir())
	p := &puller{t: t}
	deploymentID := uuid.New()
	locate := func(revisionID uuid.UUID, pull func() (string, error)) (string, error) {
		return read(cache.Locate(zap.NewNop(), deploymentID, revisionID, "", pull))
	}

	revision1 := uuid.New()
	g.Expect(locate(revision1, p.pull("first"))).To(Equal("first"))
	g.Expect(locate(revision1, p.fail)).To(Equal("first"), "a retry of the same revision must not pull")
	g.Expect(p.pulls).To(Equal(1))

	revision2 := uuid.New()
	g.Expect(locate(revision2, p.pull("overwritten"))).
		To(Equal("overwritten"), "a new revision must pull, since the version may have been overwritten")
	_, err := locate(uuid.New(), p.fail)
	g.Expect(err).To(MatchError(errOffline), "a new revision must not fall back to the chart of another revision")

	cache.Forget(deploymentID)
	_, err = locate(revision2, p.fail)
	g.Expect(err).To(MatchError(errOffline))
}
