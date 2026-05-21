package upstream

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/containers/image/v5/manifest"
	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/registry/blob"
	"github.com/distr-sh/distr/internal/registry/name"
	"github.com/distr-sh/distr/internal/tmpstream"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

type Syncer struct{}

func (s *Syncer) SyncArtifactTags(ctx context.Context, artifact *types.Artifact, skipExistingTags bool) error {
	if artifact.UpstreamURL == nil {
		return nil
	}

	log := internalctx.GetLogger(ctx).With(zap.Stringer("artifactId", artifact.ID))
	log.Debug("upstream sync started")

	repo, err := remote.NewRepository(*artifact.UpstreamURL)
	if err != nil {
		syncErr := fmt.Sprintf("failed to create upstream client: %v", err)
		return db.UpdateArtifactSyncStatus(ctx, artifact.ID, &syncErr)
	}
	repo.Client = &auth.Client{Credential: credentialForArtifact(artifact)}

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
		errStr = util.PtrTo(firstErr.Error())
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
		} else if !errors.Is(err, apierrors.ErrNotFound) {
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

	blobs, subManifests, err := extractBlobsAndSubManifests(data, contentType)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
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

	blobs, _, err := extractBlobsAndSubManifests(data, contentType)
	if err != nil {
		return fmt.Errorf("parsing sub-manifest: %w", err)
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

// FetchAndStoreBlob fetches a blob from the upstream registry for the given repo,
// stores it in the blob handler, and returns a TmpStream that can be used to serve
// the blob content to the client. The caller must call TmpStream.Destroy() after use.
// Returns apierrors.ErrNotFound if the artifact has no upstream configured.
func (s *Syncer) FetchAndStoreBlob(
	ctx context.Context,
	repoStr string,
	d godigest.Digest,
	bph blob.BlobPutHandler,
) (tmpstream.TmpStream, int64, error) {
	n, err := name.Parse(repoStr)
	if err != nil {
		return nil, 0, apierrors.ErrNotFound
	}

	artifact, err := db.GetArtifactByName(ctx, n.OrgName, n.ArtifactName)
	if err != nil || artifact.UpstreamURL == nil {
		return nil, 0, apierrors.ErrNotFound
	}

	repo, err := remote.NewRepository(*artifact.UpstreamURL)
	if err != nil {
		return nil, 0, fmt.Errorf("upstream client: %w", err)
	}
	repo.Client = &auth.Client{Credential: credentialForArtifact(artifact)}

	blobDesc, rc, err := repo.Blobs().FetchReference(ctx, d.String())
	if err != nil {
		return nil, 0, fmt.Errorf("fetching blob from upstream: %w", err)
	}
	defer rc.Close()

	tmp, err := tmpstream.New(rc)
	if err != nil {
		return nil, 0, fmt.Errorf("buffering blob: %w", err)
	}

	sr, err := tmp.Get()
	if err != nil {
		_ = tmp.Destroy()
		return nil, 0, fmt.Errorf("opening temp blob: %w", err)
	}
	if putErr := bph.Put(ctx, repoStr, d, "", sr); putErr != nil {
		sr.Close()
		_ = tmp.Destroy()
		return nil, 0, fmt.Errorf("storing blob: %w", putErr)
	}
	sr.Close()

	return tmp, blobDesc.Size, nil
}
