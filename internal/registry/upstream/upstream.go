package upstream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/containers/image/v5/manifest"
	"github.com/distr-sh/distr/internal/buildconfig"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/egress"
	"github.com/distr-sh/distr/internal/registry/name"
	"github.com/distr-sh/distr/internal/registry/verify"
	"github.com/distr-sh/distr/internal/types"
	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

var upstreamHTTPClient = sync.OnceValue(func() *http.Client {
	return &http.Client{Transport: egress.Transport(buildconfig.IsDevelopment())}
})

func ValidateUpstreamURL(ctx context.Context, upstreamURL string) error {
	repo, err := remote.NewRepository(upstreamURL)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %w", err)
	}
	if buildconfig.IsDevelopment() {
		return nil
	}
	return checkPublicRegistryHost(ctx, repo.Reference)
}

func checkPublicRegistryHost(ctx context.Context, ref registry.Reference) error {
	host := ref.Host()
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if err := egress.CheckPublicHost(ctx, strings.Trim(host, "[]")); err != nil {
		return fmt.Errorf("upstream registry must be reachable at a public address: %w", err)
	}
	return nil
}

func newUpstreamRepository(artifact *types.Artifact) (*remote.Repository, error) {
	repo, err := remote.NewRepository(*artifact.UpstreamURL)
	if err != nil {
		return nil, err
	}
	repo.Client = &auth.Client{Client: upstreamHTTPClient(), Credential: credentialForArtifact(artifact)}
	return repo, nil
}

type Syncer struct{}

func (s *Syncer) SyncArtifactTags(ctx context.Context, artifact *types.Artifact, skipExistingTags bool) error {
	if artifact.UpstreamURL == nil {
		return nil
	}

	log := internalctx.GetLogger(ctx).With(zap.Stringer("artifactId", artifact.ID))
	log.Debug("upstream sync started")

	repo, err := newUpstreamRepository(artifact)
	if err != nil {
		syncErr := fmt.Sprintf("failed to create upstream client: %v", err)
		return db.UpdateArtifactSyncStatus(ctx, artifact.ID, &syncErr)
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(5)
	tagsErr := repo.Tags(ctx, "", func(tags []string) error {
		for _, tag := range tags {
			g.Go(func() error {
				log.Debug("upstream sync tag", zap.String("tag", tag))
				if err := syncTag(gCtx, repo, artifact, tag, skipExistingTags); err != nil {
					return fmt.Errorf("syncing tag %q: %w", tag, err)
				}
				return nil
			})
		}
		return nil
	})
	syncErr := g.Wait()

	var firstErr error
	if tagsErr != nil {
		firstErr = tagsErr
	} else if syncErr != nil {
		firstErr = syncErr
	}

	var errStr *string
	if firstErr != nil {
		errStr = new(firstErr.Error())
	}

	log.Debug("upstream sync finished")

	return db.UpdateArtifactSyncStatus(ctx, artifact.ID, errStr)
}

func syncTag(
	ctx context.Context, repo *remote.Repository, artifact *types.Artifact, tag string, skipExistingTags bool,
) error {
	if skipExistingTags {
		if _, err := db.GetArtifactVersionByName(ctx, artifact.ID, tag); err == nil {
			return nil
		} else if !errors.Is(err, distrerr.ErrNotFound) {
			return fmt.Errorf("checking if tag exists: %w", err)
		}
	}
	desc, rc, err := repo.FetchReference(ctx, tag)
	if err != nil {
		return fmt.Errorf("fetching manifest: %w", err)
	}
	data, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	contentType := desc.MediaType
	if contentType == "" {
		contentType = manifest.GuessMIMEType(data)
	}
	d := desc.Digest

	blobs, subManifests, err := parseManifest(data, contentType, d)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}

	tagVersion := &types.ArtifactVersion{
		Name:                tag,
		ManifestBlobDigest:  types.Digest(d),
		ManifestBlobSize:    int64(len(data)),
		ManifestContentType: contentType,
		ManifestData:        data,
		ArtifactID:          artifact.ID,
	}
	if err := db.UpsertArtifactVersionForSync(ctx, tagVersion); err != nil {
		return err
	}

	digestVersion := &types.ArtifactVersion{
		Name:                d.String(),
		ManifestBlobDigest:  types.Digest(d),
		ManifestBlobSize:    int64(len(data)),
		ManifestContentType: contentType,
		ManifestData:        data,
		ArtifactID:          artifact.ID,
	}
	if err := db.UpsertArtifactVersionForSync(ctx, digestVersion); err != nil {
		return err
	}

	parts := make([]types.ArtifactVersionPart, 0, len(blobs)*2)
	for _, b := range blobs {
		parts = append(parts,
			types.ArtifactVersionPart{
				ArtifactVersionID:  tagVersion.ID,
				ArtifactBlobDigest: types.Digest(b.Digest),
				ArtifactBlobSize:   b.Size,
			},
			types.ArtifactVersionPart{
				ArtifactVersionID:  digestVersion.ID,
				ArtifactBlobDigest: types.Digest(b.Digest),
				ArtifactBlobSize:   b.Size,
			},
		)
	}
	if err := db.BulkUpsertArtifactVersionParts(ctx, parts); err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(5)
	for _, sub := range subManifests {
		g.Go(func() error {
			if err := syncSubManifest(ctx, repo, artifact, sub); err != nil {
				return fmt.Errorf("syncing sub-manifest %s: %w", sub.Digest, err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func syncSubManifest(
	ctx context.Context, repo *remote.Repository, artifact *types.Artifact, desc ocispec.Descriptor,
) error {
	rc, err := repo.Fetch(ctx, desc)
	if err != nil {
		return fmt.Errorf("fetching sub-manifest: %w", err)
	}
	data, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return fmt.Errorf("reading sub-manifest: %w", err)
	}

	contentType := desc.MediaType
	if contentType == "" {
		contentType = manifest.GuessMIMEType(data)
	}
	d := godigest.Digest(desc.Digest.String())

	blobs, _, err := parseManifest(data, contentType, d)
	if err != nil {
		return fmt.Errorf("parsing sub-manifest: %w", err)
	}

	version := &types.ArtifactVersion{
		Name:                d.String(),
		ManifestBlobDigest:  types.Digest(d),
		ManifestBlobSize:    int64(len(data)),
		ManifestContentType: contentType,
		ManifestData:        data,
		ArtifactID:          artifact.ID,
	}
	if err := db.UpsertArtifactVersionForSync(ctx, version); err != nil {
		return err
	}

	parts := make([]types.ArtifactVersionPart, len(blobs))
	for i, b := range blobs {
		parts[i] = types.ArtifactVersionPart{
			ArtifactVersionID:  version.ID,
			ArtifactBlobDigest: types.Digest(b.Digest),
			ArtifactBlobSize:   b.Size,
		}
	}
	return db.BulkUpsertArtifactVersionParts(ctx, parts)
}

type blobRef struct {
	Digest godigest.Digest
	Size   int64
}

func parseManifest(
	data []byte, contentType string, d godigest.Digest,
) (blobs []blobRef, subManifests []ocispec.Descriptor, err error) {
	if err := d.Validate(); err != nil {
		return nil, nil, fmt.Errorf("invalid manifest digest: %w", err)
	}
	if d.Algorithm().FromBytes(data) != d {
		return nil, nil, fmt.Errorf("manifest content does not match its digest %v", d)
	}
	return extractBlobsAndSubManifests(data, contentType)
}

func extractBlobsAndSubManifests(
	data []byte, contentType string,
) (blobs []blobRef, subManifests []ocispec.Descriptor, err error) {
	if manifest.MIMETypeIsMultiImage(contentType) {
		im, err := manifest.ListFromBlob(data, contentType)
		if err != nil {
			return nil, nil, err
		}
		for _, d := range im.Instances() {
			i, err := im.Instance(d)
			if err != nil {
				return nil, nil, err
			}
			subManifests = append(subManifests, ocispec.Descriptor{
				MediaType: i.MediaType,
				Digest:    d,
				Size:      i.Size,
			})
			blobs = append(blobs, blobRef{Digest: d, Size: i.Size})
		}
	} else {
		m, err := manifest.FromBlob(data, contentType)
		if err != nil {
			return nil, nil, err
		}
		c := m.ConfigInfo()
		blobs = append(blobs, blobRef{Digest: c.Digest, Size: c.Size})
		for _, l := range m.LayerInfos() {
			blobs = append(blobs, blobRef{Digest: l.Digest, Size: l.Size})
		}
	}
	return blobs, subManifests, nil
}

// OpenBlob opens the blob identified by d at the upstream registry of repoStr and returns its size and content.
// The content ends with an error instead of io.EOF if it does not match d, since upstream registries are not
// trusted to serve what they announce. The download stays bound to ctx until the caller closes the content.
// Returns distrerr.ErrNotFound if the artifact has no upstream configured.
func (s *Syncer) OpenBlob(ctx context.Context, repoStr string, d godigest.Digest) (int64, io.ReadCloser, error) {
	n, err := name.Parse(repoStr)
	if err != nil {
		return 0, nil, distrerr.ErrNotFound
	}

	artifact, err := db.GetArtifactByName(ctx, n.OrgName, n.ArtifactName)
	if err != nil {
		return 0, nil, fmt.Errorf("getting artifact by name: %w", err)
	} else if artifact.UpstreamURL == nil {
		return 0, nil, distrerr.ErrNotFound
	}

	repo, err := newUpstreamRepository(artifact)
	if err != nil {
		return 0, nil, fmt.Errorf("upstream client: %w", err)
	}

	desc, rc, err := repo.Blobs().FetchReference(ctx, d.String())
	if err != nil {
		return 0, nil, fmt.Errorf("fetching blob from upstream: %w", err)
	}

	vrc, err := verify.ReadCloser(rc, desc.Size, d)
	if err != nil {
		_ = rc.Close()
		return 0, nil, err
	}
	return desc.Size, vrc, nil
}
