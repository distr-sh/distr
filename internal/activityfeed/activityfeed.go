// Package activityfeed assembles the activity feed of one or more customer organizations from the records the
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
	// CustomerOrganizationID is nil for activity of the vendor that concerns no particular customer
	// organization, such as a new application version.
	CustomerOrganizationID   *uuid.UUID
	CustomerOrganizationName *string
}

func newUser(name *string, imageID *uuid.UUID) *User {
	if name == nil {
		return nil
	}
	return &User{Name: *name, ImageID: imageID}
}

// RestrictToCustomerOrganizations returns the entries with the name of their customer organization. Entries of
// a customer organization outside customerOrgs are left out, so that a source returning more than it was asked
// for cannot widen what the caller sees.
func RestrictToCustomerOrganizations(entries []Entry, customerOrgs []types.CustomerOrganization) []Entry {
	names := make(map[uuid.UUID]string, len(customerOrgs))
	for _, customerOrg := range customerOrgs {
		names[customerOrg.ID] = customerOrg.Name
	}
	result := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.CustomerOrganizationID == nil {
			result = append(result, entry)
		} else if name, ok := names[*entry.CustomerOrganizationID]; ok {
			entry.CustomerOrganizationName = &name
			result = append(result, entry)
		}
	}
	return result
}

// Latest returns the newest entries, at most Limit of them.
func Latest(entries []Entry) []Entry {
	slices.SortFunc(entries, func(a, b Entry) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return entries[:min(len(entries), Limit)]
}

func ApplicationVersions(versions []types.ApplicationVersionEvent) []Entry {
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

// ArtifactTags returns an entry for every tag, which has to belong to one of the artifacts.
func ArtifactTags(artifacts []types.ArtifactWithDownloads, tags []types.ArtifactVersion) []Entry {
	names := make(map[uuid.UUID]string, len(artifacts))
	for _, artifact := range artifacts {
		names[artifact.ID] = artifact.Name
	}
	result := make([]Entry, len(tags))
	for i, tag := range tags {
		result[i] = Entry{
			ID:         tag.ID,
			CreatedAt:  tag.CreatedAt,
			Type:       EntryTypeArtifactVersionCreated,
			ResourceID: tag.ArtifactID,
			Subject:    names[tag.ArtifactID],
			Detail:     new(tag.Name),
		}
	}
	return result
}

// FirstPulls returns an entry for every pull, which has to have its customer organization set.
func FirstPulls(pulls []types.ArtifactVersionPull) []Entry {
	result := make([]Entry, len(pulls))
	for i, pull := range pulls {
		result[i] = Entry{
			ID:                     pull.ID,
			CreatedAt:              pull.CreatedAt,
			Type:                   EntryTypeArtifactVersionFirstPulled,
			ResourceID:             pull.Artifact.ID,
			Subject:                pull.Artifact.Name,
			Detail:                 new(pull.ArtifactVersion.Name),
			CustomerOrganizationID: new(pull.CustomerOrganization.ID),
		}
		if pull.UserAccount != nil {
			result[i].User = &User{Name: pull.UserAccount.Name, ImageID: pull.UserAccount.ImageID}
		}
	}
	return result
}

func DeploymentRevisions(revisions []types.DeploymentRevisionEvent) []Entry {
	result := make([]Entry, len(revisions))
	for i, revision := range revisions {
		result[i] = Entry{
			ID:                     revision.ID,
			CreatedAt:              revision.CreatedAt,
			Type:                   EntryTypeDeploymentRevisionCreated,
			ResourceID:             revision.DeploymentTargetID,
			Subject:                revision.ApplicationName + " (" + revision.DeploymentTargetName + ")",
			Detail:                 new(revision.ApplicationVersionName),
			User:                   newUser(revision.CreatedByName, revision.CreatedByImageID),
			CustomerOrganizationID: revision.CustomerOrganizationID,
		}
	}
	return result
}

// LicenseKeys returns an entry for the latest revision of every key. A key without a license template is created
// together with its first revision in one transaction, so the two share a timestamp until the key is revised
// again. The first revision of a key with a template comes later and counts as an update.
func LicenseKeys(keys []types.LicenseKeyEvent) []Entry {
	result := make([]Entry, len(keys))
	for i, key := range keys {
		entryType := EntryTypeLicenseKeyUpdated
		if key.LastRevisedAt.Equal(key.CreatedAt) {
			entryType = EntryTypeLicenseKeyCreated
		}
		result[i] = Entry{
			ID:                     key.ID,
			CreatedAt:              key.LastRevisedAt,
			Type:                   entryType,
			ResourceID:             key.ID,
			Subject:                key.Name,
			CustomerOrganizationID: new(key.CustomerOrganizationID),
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
			ID:                     event.SupportBundleID,
			CreatedAt:              event.CreatedAt,
			Type:                   entryType,
			ResourceID:             event.SupportBundleID,
			Subject:                event.Title,
			User:                   newUser(event.UserName, event.UserImageID),
			CustomerOrganizationID: new(event.CustomerOrganizationID),
		})
	}
	return result
}
