package db

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/notification"
	"github.com/distr-sh/distr/internal/registry/manifest"
	"github.com/distr-sh/distr/internal/registry/name"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
)

type handler struct{}

func NewManifestHandler() manifest.ManifestHandler {
	return &handler{}
}

// entitlementCustomerOrgID returns the customer organization whose entitlements narrow down what a
// listing may show, and nil for a vendor user and for an anonymous request, which reaches nothing
// but a public artifact. A public artifact is not narrowed either, which the queries decide.
func entitlementCustomerOrgID(ctx context.Context) *uuid.UUID {
	if principal, ok := auth.ArtifactsPrincipal(ctx); !ok {
		return nil
	} else if principal.CurrentOrg().HasFeature(types.FeatureLicensing) {
		return principal.CurrentCustomerOrgID()
	} else {
		return nil
	}
}

// Delete removes a manifest by tag reference.
func (h *handler) Delete(ctx context.Context, nameStr string, reference string) error {
	if _, err := digest.Parse(reference); err == nil {
		return distrerr.NewBadRequest("digest-based deletion is not supported, delete by tag instead")
	}

	n, err := name.Parse(nameStr)
	if err != nil {
		return fmt.Errorf("%w: %w", manifest.ErrNameUnknown, err)
	}

	artifact, err := db.GetArtifactByName(ctx, n.OrgName, n.ArtifactName)
	if err != nil {
		if errors.Is(err, distrerr.ErrNotFound) {
			return fmt.Errorf("%w: %w", manifest.ErrNameUnknown, err)
		}
		return err
	}

	if err := db.DeleteArtifactTag(ctx, artifact.ID, reference); err != nil {
		if errors.Is(err, distrerr.ErrNotFound) {
			return fmt.Errorf("%w: %w", manifest.ErrManifestUnknown, err)
		}
		return err
	}
	return nil
}

// Get implements manifest.ManifestHandler.
func (h *handler) Get(ctx context.Context, nameStr string, reference string) (*manifest.Manifest, error) {
	if name, err := name.Parse(nameStr); err != nil {
		return nil, fmt.Errorf("%w: %w", manifest.ErrNameUnknown, err)
	} else if av, err := db.GetArtifactVersion(ctx, name.OrgName, name.ArtifactName, reference); err != nil {
		if errors.Is(err, distrerr.ErrNotFound) {
			return nil, fmt.Errorf("%w: %w", manifest.ErrNameUnknown, err)
		}
		return nil, err
	} else {
		return &manifest.Manifest{
			BlobWithData: manifest.BlobWithData{
				Blob: manifest.Blob{
					Digest: digest.Digest(av.ManifestBlobDigest),
					Size:   av.ManifestBlobSize,
				},
				Data: av.ManifestData,
			},
			ContentType: av.ManifestContentType,
		}, nil
	}
}

// List implements manifest.ManifestHandler.
func (h *handler) List(ctx context.Context, n int) ([]string, error) {
	auth := auth.ArtifactsAuthentication.Require(ctx)
	var artifacts []types.ArtifactWithDownloads
	var err error
	if auth.CurrentOrg().HasFeature(types.FeatureLicensing) && auth.CurrentCustomerOrgID() != nil {
		if entitlements, err1 := db.GetArtifactEntitlements(ctx, *auth.CurrentOrgID()); err1 != nil {
			err = err1
		} else if len(entitlements) > 0 {
			artifacts, err = db.GetArtifactsByEntitlementOwnerID(ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID())
		} else {
			artifacts, err = db.GetArtifactsByOrgID(ctx, *auth.CurrentOrgID())
		}
	} else {
		artifacts, err = db.GetArtifactsByOrgID(ctx, *auth.CurrentOrgID())
	}
	if err != nil {
		return nil, err
	}
	result := make([]string, len(artifacts))
	for i, artifact := range artifacts {
		name := name.Name{OrgName: artifact.OrganizationSlug, ArtifactName: artifact.Name}
		result[i] = name.String()
	}
	// TODO: move to DB
	if 0 < n && n < len(result) {
		result = result[:n]
	}
	return result, nil
}

// ListDigests implements manifest.ManifestHandler.
func (h *handler) ListDigests(ctx context.Context, nameStr string) ([]digest.Digest, error) {
	if name, err := name.Parse(nameStr); err != nil {
		return nil, fmt.Errorf("%w: %w", manifest.ErrNameUnknown, err)
	} else {
		if artifact, err := db.GetArtifactByName(ctx, name.OrgName, name.ArtifactName); err != nil {
			if errors.Is(err, distrerr.ErrNotFound) {
				return nil, fmt.Errorf("%w: %w", manifest.ErrNameUnknown, err)
			}
			return nil, err
		} else if versions, err := db.GetVersionsForArtifact(
			ctx,
			artifact.ID,
			entitlementCustomerOrgID(ctx),
		); err != nil {
			return nil, err
		} else {
			var result []digest.Digest
			for _, version := range versions {
				if h, err := digest.Parse(version.Digest); err != nil {
					continue
				} else {
					result = append(result, h)
				}
			}
			return result, nil
		}
	}
}

// ListTags implements manifest.ManifestHandler.
func (h *handler) ListTags(ctx context.Context, nameStr string, n int, last string) ([]string, error) {
	if name, err := name.Parse(nameStr); err != nil {
		return nil, fmt.Errorf("%w: %w", manifest.ErrNameUnknown, err)
	} else {
		if artifact, err := db.GetArtifactByName(ctx, name.OrgName, name.ArtifactName); err != nil {
			if errors.Is(err, distrerr.ErrNotFound) {
				return nil, fmt.Errorf("%w: %w", manifest.ErrNameUnknown, err)
			}
			return nil, err
		} else if versions, err := db.GetVersionsForArtifact(
			ctx,
			artifact.ID,
			entitlementCustomerOrgID(ctx),
		); err != nil {
			return nil, err
		} else {
			result := []string{}
			for _, version := range versions {
				for _, tag := range version.Tags {
					if tag.Name > last {
						result = append(result, tag.Name)
					}
				}
			}
			slices.Sort(result)
			if len(result) > n {
				result = result[:n]
			}
			return result, nil
		}
	}
}

// Put implements manifest.ManifestHandler.
func (h *handler) Put(
	ctx context.Context,
	nameStr, reference string,
	manifestData manifest.Manifest,
	blobs []manifest.Blob,
) error {
	auth := auth.ArtifactsAuthentication.Require(ctx)
	name, err := name.Parse(nameStr)
	if err != nil {
		return err
	}

	// A push of a tag that already exists with the same content leaves this nil, so that only an
	// actually created version is announced.
	var created *types.ArtifactVersion

	if err := db.RunTx(ctx, func(ctx context.Context) error {
		artifact, err := db.GetOrCreateArtifact(ctx, *auth.CurrentOrgID(), name.ArtifactName)
		if err != nil {
			return err
		}

		version := types.ArtifactVersion{
			CreatedByUserAccountID: new(auth.CurrentUserID()),
			Name:                   reference,
			ManifestBlobDigest:     types.Digest(manifestData.Digest),
			ManifestBlobSize:       manifestData.Size,
			ManifestContentType:    manifestData.ContentType,
			ManifestData:           manifestData.Data,
			ArtifactID:             artifact.ID,
		}

		if existingVersion, err := db.GetArtifactVersion(ctx, name.OrgName, name.ArtifactName, reference); err != nil {
			if !errors.Is(err, distrerr.ErrNotFound) {
				return err
			} else if quotaOk, err := db.EnsureArtifactTagLimitForInsert(ctx, *auth.CurrentOrgID()); err != nil {
				return err
			} else if !quotaOk {
				return distrerr.ErrQuotaExceeded
			}
		} else if existingVersion.ManifestBlobDigest == version.ManifestBlobDigest &&
			existingVersion.ManifestContentType == version.ManifestContentType {
			// Tag already exists with the same content: nothing to do
			return nil
		} else if !auth.CurrentOrg().HasFeature(types.FeatureArtifactVersionMutable) {
			return fmt.Errorf("%w: tag %s already exists with different content", manifest.ErrTagAlreadyExists, reference)
		} else if err := db.DeleteArtifactVersion(ctx, existingVersion.ArtifactID, existingVersion.Name); err != nil {
			return err
		}

		if err := db.CreateArtifactVersion(ctx, &version); err != nil {
			return err
		}

		for _, blob := range blobs {
			part := types.ArtifactVersionPart{
				ArtifactVersionID:  version.ID,
				ArtifactBlobDigest: types.Digest(blob.Digest),
				ArtifactBlobSize:   blob.Size,
			}
			if err := db.CreateArtifactVersionPart(ctx, &part); err != nil {
				return err
			}
		}
		created = &version
		return addTagToMatchingEntitlements(ctx, version)
	}); err != nil {
		return err
	}

	if created != nil {
		db.RunAfterTx(ctx, func(ctx context.Context) {
			notification.Dispatch(ctx, func(ctx context.Context) error {
				return notification.SendArtifactVersionAvailableNotifications(ctx, *created)
			})
		})
	}

	return nil
}

// addTagToMatchingEntitlements adds the digest version of a new tag, which is what an entitlement selects, to every
// entitlement whose auto-add regex matches the tag. It runs in the transaction that creates the tag, so that the
// notifications sent after it already see the tag as entitled.
func addTagToMatchingEntitlements(ctx context.Context, tag types.ArtifactVersion) error {
	if tag.IsDigestVersion() {
		return nil
	}
	// Without the lock, an entitlement switched to all tags concurrently would end up with only this version, because
	// the insert below does not wait for the edit that removes its versions.
	if err := db.LockArtifactRows(ctx, []uuid.UUID{tag.ArtifactID}); err != nil {
		return err
	}
	entitlements, err := db.GetArtifactEntitlementsWithAutoAddVersionRegex(ctx, tag.ArtifactID)
	if err != nil {
		return err
	}
	var entitlementIDs []uuid.UUID
	for _, entitlement := range entitlements {
		if re, err := regexp.Compile(*entitlement.AutoAddVersionRegex); err != nil {
			return fmt.Errorf("could not compile auto-add version regex of artifact entitlement %v: %w", entitlement.ID, err)
		} else if re.MatchString(tag.Name) {
			entitlementIDs = append(entitlementIDs, entitlement.ID)
		}
	}
	if len(entitlementIDs) == 0 {
		return nil
	}
	return db.AddVersionToArtifactEntitlements(ctx, tag.ArtifactID, tag.ManifestBlobDigest, entitlementIDs)
}
