// Package chartcache keeps the Helm chart of the last revision of every deployment, so that the Kubernetes
// controller neither pulls the chart again while it retries a revision nor fails to apply a revision that only
// changes values while the chart's registry is unreachable.
package chartcache

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	chartFileName = "chart.tgz"
	entryFileName = "entry.json"
)

type Chart struct {
	URL     string `json:"url"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Digest  string `json:"digest,omitempty"`
}

type entry struct {
	RevisionID uuid.UUID `json:"revisionId"`
	Chart      Chart     `json:"chart"`
}

type Cache struct {
	Dir string
}

// Locate returns the path of the chart archive for a revision of a deployment. It calls pull, which returns the
// path of a freshly pulled archive, unless the cached chart has the same digest or, for a chart without a digest,
// was cached for the same revision. A new revision of a chart without a digest always pulls, because a mutable
// artifact version may have been overwritten since, and falls back to the cached chart only if the pull fails and
// the chart reference is unchanged.
func (c Cache) Locate(
	logger *zap.Logger,
	deploymentID uuid.UUID,
	revisionID uuid.UUID,
	chart Chart,
	pull func() (string, error),
) (string, error) {
	cached, err := c.readEntry(deploymentID)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		logger.Warn("could not read chart cache entry", zap.Error(err))
	}
	cachedChartMatches := cached != nil && cached.Chart == chart

	if cachedChartMatches && (chart.Digest != "" || cached.RevisionID == revisionID) {
		logger.Debug("using cached chart")
		return c.chartPath(deploymentID), nil
	}

	pulledPath, pullErr := pull()
	if pullErr != nil {
		if !cachedChartMatches {
			return "", pullErr
		}
		logger.Warn("could not pull chart, using cached chart", zap.Error(pullErr))
		pulledPath = ""
	}

	if err := c.write(deploymentID, entry{RevisionID: revisionID, Chart: chart}, pulledPath); err != nil {
		logger.Warn("could not write chart cache entry", zap.Error(err))
		if pulledPath != "" {
			return pulledPath, nil
		}
	}
	return c.chartPath(deploymentID), nil
}

func (c Cache) Delete(deploymentID uuid.UUID) error {
	return os.RemoveAll(c.entryDir(deploymentID))
}

func (c Cache) entryDir(deploymentID uuid.UUID) string {
	return filepath.Join(c.Dir, deploymentID.String())
}

func (c Cache) chartPath(deploymentID uuid.UUID) string {
	return filepath.Join(c.entryDir(deploymentID), chartFileName)
}

func (c Cache) readEntry(deploymentID uuid.UUID) (*entry, error) {
	if _, err := os.Stat(c.chartPath(deploymentID)); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(c.entryDir(deploymentID), entryFileName))
	if err != nil {
		return nil, err
	}
	var e entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// write replaces the chart archive with the one at chartPath, unless it is empty, and then the entry. The entry is
// removed before the archive is replaced, so that a crash never leaves an entry describing another archive.
func (c Cache) write(deploymentID uuid.UUID, e entry, chartPath string) error {
	dir := c.entryDir(deploymentID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if chartPath != "" {
		if err := os.Remove(filepath.Join(dir, entryFileName)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if src, err := os.Open(chartPath); err != nil {
			return err
		} else {
			err := writeFileAtomic(filepath.Join(dir, chartFileName), src)
			_ = src.Close()
			if err != nil {
				return err
			}
		}
	}
	if data, err := json.Marshal(e); err != nil {
		return err
	} else {
		return writeFileAtomic(filepath.Join(dir, entryFileName), bytes.NewReader(data))
	}
}

func writeFileAtomic(path string, r io.Reader) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("could not write %v: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
