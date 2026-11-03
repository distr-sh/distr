// Package chartcache keeps the Kubernetes controller from pulling a Helm chart again that it already has, since a
// failing revision is retried on every loop and would otherwise pull its chart each time.
package chartcache

import (
	"encoding/hex"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
	"go.uber.org/zap"
	"helm.sh/helm/v4/pkg/downloader"
)

type Cache struct {
	content downloader.Cache
	mu      sync.Mutex
	pulled  map[uuid.UUID]pulledChart
}

type pulledChart struct {
	revisionID uuid.UUID
	path       string
}

// New returns a cache that stores charts with a digest in Helm's content cache at contentDir, where they outlive the
// controller process, and remembers the chart of every other revision in memory.
func New(contentDir string) *Cache {
	return &Cache{content: &downloader.DiskCache{Root: contentDir}, pulled: map[uuid.UUID]pulledChart{}}
}

// Locate returns the path of the chart archive for a revision of a deployment. It calls pull, which returns the path
// of a freshly pulled archive, unless the content cache holds the chart's digest or, for a chart without a digest,
// the chart was pulled for the same revision before.
func (c *Cache) Locate(
	logger *zap.Logger,
	deploymentID uuid.UUID,
	revisionID uuid.UUID,
	chartDigest string,
	pull func() (string, error),
) (string, error) {
	if chartDigest != "" {
		if key, err := contentKey(chartDigest); err != nil {
			logger.Warn("could not use chart digest as cache key", zap.Error(err))
		} else {
			return c.locateByDigest(logger, key, pull)
		}
	}

	c.mu.Lock()
	cached, ok := c.pulled[deploymentID]
	c.mu.Unlock()
	if ok && cached.revisionID == revisionID {
		if _, err := os.Stat(cached.path); err == nil {
			return cached.path, nil
		}
	}

	path, err := pull()
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.pulled[deploymentID] = pulledChart{revisionID: revisionID, path: path}
	c.mu.Unlock()
	return path, nil
}

func (c *Cache) Forget(deploymentID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pulled, deploymentID)
}

// locateByDigest stores the pulled archive under the manifest digest itself, since Helm stores a chart pulled by
// digest under the hash of the archive and so never finds it by the manifest digest again.
func (c *Cache) locateByDigest(
	logger *zap.Logger,
	key [32]byte,
	pull func() (string, error),
) (string, error) {
	if path, err := c.content.Get(key, downloader.CacheChart); err == nil {
		return path, nil
	}
	pulledPath, err := pull()
	if err != nil {
		return "", err
	}
	f, err := os.Open(pulledPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	if path, err := c.content.Put(key, f, downloader.CacheChart); err != nil {
		logger.Warn("could not put chart into cache", zap.Error(err))
		return pulledPath, nil
	} else {
		return path, nil
	}
}

func contentKey(chartDigest string) ([32]byte, error) {
	var key [32]byte
	d, err := digest.Parse(chartDigest)
	if err != nil {
		return key, err
	} else if d.Algorithm() != digest.SHA256 {
		return key, digest.ErrDigestUnsupported
	}
	b, err := hex.DecodeString(d.Encoded())
	copy(key[:], b)
	return key, err
}
