package chartcache_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/distr-sh/distr/internal/chartcache"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

func TestLocate(t *testing.T) {
	g := NewWithT(t)
	cache := chartcache.Cache{Dir: t.TempDir()}
	deploymentID := uuid.New()
	chart := chartcache.Chart{URL: "oci://example.com/charts/app", Name: "app", Version: "1.0.0"}
	errOffline := errors.New("registry unreachable")

	pulls := 0
	pull := func(content string) func() (string, error) {
		return func() (string, error) {
			pulls++
			path := filepath.Join(t.TempDir(), "app-1.0.0.tgz")
			return path, os.WriteFile(path, []byte(content), 0o600)
		}
	}
	pullFails := func() (string, error) {
		pulls++
		return "", errOffline
	}
	locate := func(revisionID uuid.UUID, chart chartcache.Chart, pull func() (string, error)) (string, error) {
		path, err := cache.Locate(zap.NewNop(), deploymentID, revisionID, chart, pull)
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(path)
		return string(data), err
	}

	revision1 := uuid.New()
	g.Expect(locate(revision1, chart, pull("first"))).To(Equal("first"))
	g.Expect(pulls).To(Equal(1))

	g.Expect(locate(revision1, chart, pull("ignored"))).To(Equal("first"), "a retry of the same revision must not pull")
	g.Expect(pulls).To(Equal(1))

	revision2 := uuid.New()
	g.Expect(locate(revision2, chart, pull("overwritten"))).
		To(Equal("overwritten"), "a new revision must pull, since the version may have been overwritten")
	g.Expect(pulls).To(Equal(2))

	revision3 := uuid.New()
	g.Expect(locate(revision3, chart, pullFails)).To(Equal("overwritten"), "a failed pull falls back to the cache")
	g.Expect(locate(revision3, chart, pullFails)).To(Equal("overwritten"))
	g.Expect(pulls).To(Equal(3), "a retry after the fallback must not pull")

	otherVersion := chart
	otherVersion.Version = "2.0.0"
	_, err := locate(uuid.New(), otherVersion, pullFails)
	g.Expect(err).To(MatchError(errOffline), "the cached chart must not stand in for another version")

	g.Expect(locate(revision3, chart, pull("ignored"))).To(Equal("overwritten"), "the cache survives a failed pull")

	g.Expect(cache.Delete(deploymentID)).To(Succeed())
	_, err = locate(revision3, chart, pullFails)
	g.Expect(err).To(MatchError(errOffline))
}

func TestLocateWithDigest(t *testing.T) {
	g := NewWithT(t)
	cache := chartcache.Cache{Dir: t.TempDir()}
	deploymentID := uuid.New()
	chart := chartcache.Chart{URL: "oci://example.com/charts/app", Version: "1.0.0", Digest: "sha256:aaaa"}
	errOffline := errors.New("registry unreachable")

	pulls := 0
	pull := func(content string) func() (string, error) {
		return func() (string, error) {
			pulls++
			path := filepath.Join(t.TempDir(), "app.tgz")
			return path, os.WriteFile(path, []byte(content), 0o600)
		}
	}
	pullFails := func() (string, error) {
		pulls++
		return "", errOffline
	}
	locate := func(chart chartcache.Chart, pull func() (string, error)) (string, error) {
		path, err := cache.Locate(zap.NewNop(), deploymentID, uuid.New(), chart, pull)
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(path)
		return string(data), err
	}

	g.Expect(locate(chart, pull("a"))).To(Equal("a"))
	g.Expect(locate(chart, pullFails)).To(Equal("a"), "a new revision with the same digest must not pull")
	g.Expect(pulls).To(Equal(1))

	overwritten := chart
	overwritten.Digest = "sha256:bbbb"
	_, err := locate(overwritten, pullFails)
	g.Expect(err).To(MatchError(errOffline), "the cached chart must not stand in for another digest")
	g.Expect(locate(overwritten, pull("b"))).To(Equal("b"))
}
