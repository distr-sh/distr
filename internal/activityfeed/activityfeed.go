// Package activityfeed assembles the activity feed of a customer organization from the records the
// existing queries return.
package activityfeed

import (
	"slices"
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

const (
	Window = 90 * 24 * time.Hour
	Limit  = 25
)

type EntryType string

const (
	EntryTypeApplicationVersionCreated  EntryType = "application_version_created"
	EntryTypeArtifactVersionCreated     EntryType = "artifact_version_created"
	EntryTypeArtifactVersionFirstPulled EntryType = "artifact_version_first_pulled"
	EntryTypeDeploymentRevisionCreated  EntryType = "deployment_revision_created"
	EntryTypeLicenseKeyCreated          EntryType = "license_key_created"
	EntryTypeLicenseKeyUpdated          EntryType = "license_key_updated"
	EntryTypeSupportBundleCreated       EntryType = "support_bundle_created"
	EntryTypeSupportBundleUploaded      EntryType = "support_bundle_uploaded"
	EntryTypeSupportBundleResolved      EntryType = "support_bundle_resolved"
	EntryTypeSupportBundleCanceled      EntryType = "support_bundle_canceled"
)

type User struct {
	Name    string
	ImageID *uuid.UUID
}

type Entry struct {
	ID        uuid.UUID
	CreatedAt time.Time
	Type      EntryType
	// ResourceID is the application, artifact, deployment target, license key or support bundle the
	// entry is about, depending on its type.
	ResourceID uuid.UUID
	Subject    string
	Detail     *string
	User       *User
}

func newUser(name *string, imageID *uuid.UUID) *User {
	if name == nil {
		return nil
	}
	return &User{Name: *name, ImageID: imageID}
}

// Latest returns the newest entries, at most Limit of them.
func Latest(entries []Entry) []Entry {
	slices.SortFunc(entries, func(a, b Entry) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return entries[:min(len(entries), Limit)]
}

func ApplicationVersions(versions []types.CustomerApplicationVersion) []Entry {
	result := make([]Entry, len(versions))
	for i, version := range versions {
		result[i] = Entry{
			ID:         version.ID,
			CreatedAt:  version.CreatedAt,
			Type:       EntryTypeApplicationVersionCreated,
			ResourceID: version.ApplicationID,
			Subject:    version.ApplicationName,
			Detail:     new(version.Name),
		}
	}
	return result
}

// ArtifactVersions returns the versions of the artifact published since that carry a release tag.
// versions have to be the versions the customer may pull.
func ArtifactVersions(
	artifact types.ArtifactWithDownloads,
	versions []types.TaggedArtifactVersion,
	since time.Time,
) []Entry {
	var result []Entry
	for _, version := range versions {
		if !version.CreatedAt.After(since) {
			continue
		} else if _, tag := types.LatestArtifactTag([]types.TaggedArtifactVersion{version}); tag != "" {
			result = append(result, Entry{
				ID:         version.ID,
				CreatedAt:  version.CreatedAt,
				Type:       EntryTypeArtifactVersionCreated,
				ResourceID: artifact.ID,
				Subject:    artifact.Name,
				Detail:     new(tag),
			})
		}
	}
	return result
}

func FirstPulls(pulls []types.ArtifactVersionPull) []Entry {
	result := make([]Entry, len(pulls))
	for i, pull := range pulls {
		result[i] = Entry{
			ID:         pull.ID,
			CreatedAt:  pull.CreatedAt,
			Type:       EntryTypeArtifactVersionFirstPulled,
			ResourceID: pull.Artifact.ID,
			Subject:    pull.Artifact.Name,
			Detail:     new(pull.ArtifactVersion.Name),
		}
		if pull.UserAccount != nil {
			result[i].User = &User{Name: pull.UserAccount.Name, ImageID: pull.UserAccount.ImageID}
		}
	}
	return result
}

func DeploymentRevisions(revisions []types.CustomerDeploymentRevision) []Entry {
	result := make([]Entry, len(revisions))
	for i, revision := range revisions {
		result[i] = Entry{
			ID:         revision.ID,
			CreatedAt:  revision.CreatedAt,
			Type:       EntryTypeDeploymentRevisionCreated,
			ResourceID: revision.DeploymentTargetID,
			Subject:    revision.ApplicationName,
			Detail:     new(revision.ApplicationVersionName + " on " + revision.DeploymentTargetName),
			User:       newUser(revision.CreatedByName, revision.CreatedByImageID),
		}
	}
	return result
}

func LicenseKeyRevisions(revisions []types.CustomerLicenseKeyRevision) []Entry {
	result := make([]Entry, len(revisions))
	for i, revision := range revisions {
		entryType := EntryTypeLicenseKeyCreated
		if revision.IsUpdate {
			entryType = EntryTypeLicenseKeyUpdated
		}
		result[i] = Entry{
			ID:         revision.ID,
			CreatedAt:  revision.CreatedAt,
			Type:       entryType,
			ResourceID: revision.LicenseKeyID,
			Subject:    revision.LicenseKeyName,
		}
	}
	return result
}

// SupportBundles skips status changes that have no entry type, which is a status a bundle is created in.
func SupportBundles(events []types.CustomerSupportBundleEvent) []Entry {
	result := make([]Entry, 0, len(events))
	for _, event := range events {
		entryType := EntryTypeSupportBundleCreated
		if event.Status != nil {
			switch *event.Status {
			case types.SupportBundleStatusCreated:
				entryType = EntryTypeSupportBundleUploaded
			case types.SupportBundleStatusResolved:
				entryType = EntryTypeSupportBundleResolved
			case types.SupportBundleStatusCanceled:
				entryType = EntryTypeSupportBundleCanceled
			default:
				continue
			}
		}
		result = append(result, Entry{
			ID:         event.SupportBundleID,
			CreatedAt:  event.CreatedAt,
			Type:       entryType,
			ResourceID: event.SupportBundleID,
			Subject:    event.Title,
			User:       newUser(event.UserName, event.UserImageID),
		})
	}
	return result
}
