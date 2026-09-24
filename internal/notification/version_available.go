package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mailsending"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SendTimeout bounds a send that is deferred into a goroutine and therefore outlives the request it
// belongs to.
const SendTimeout = 30 * time.Second

// SendApplicationUpdateAvailableNotifications tells the recipients of every configuration that
// watches the version's application that it exists, listing the deployments they may see that do
// not run it yet.
func SendApplicationUpdateAvailableNotifications(
	ctx context.Context,
	version types.ApplicationVersion,
) error {
	configs, err := db.GetUpdateNotificationConfigurationsForApplication(ctx, version.ApplicationID)
	if err != nil {
		return fmt.Errorf("failed to get update notification configurations: %w", err)
	} else if len(configs) == 0 {
		return nil
	}

	deployments, err := db.GetDeploymentsPendingUpdate(ctx, version.ID)
	if err != nil {
		return fmt.Errorf("failed to get deployments pending update: %w", err)
	} else if len(deployments) == 0 {
		return nil
	}

	// A configuration can only link applications of its own organization, so any of them names
	// the organization the application belongs to.
	application, err := db.GetApplication(ctx, version.ApplicationID, configs[0].OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get application: %w", err)
	}
	if deployments = deploymentsBehind(*application, version, deployments); len(deployments) == 0 {
		return nil
	}

	var aggErr error
	for _, config := range configs {
		if err := sendApplicationUpdateAvailableWithConfig(ctx, config, version, deployments); err != nil {
			aggErr = errors.Join(aggErr, fmt.Errorf("config %v: %w", config.ID, err))
		}
	}
	return aggErr
}

// SendApplicationEntitlementVersionsNotifications announces the newest of the versions an
// entitlement was just widened to. A customer whose entitlement did not cover a version when it
// was created is skipped then, so this is the second and last moment at which they can learn about
// it. Recipients who were notified at creation time are held back by their notification records.
func SendApplicationEntitlementVersionsNotifications(
	ctx context.Context,
	applicationVersionIDs []uuid.UUID,
) error {
	version, err := db.GetNewestApplicationVersion(ctx, applicationVersionIDs)
	if err != nil {
		return fmt.Errorf("failed to get newest application version: %w", err)
	} else if version == nil {
		return nil
	}
	return SendApplicationUpdateAvailableNotifications(ctx, *version)
}

func sendApplicationUpdateAvailableWithConfig(
	ctx context.Context,
	config types.UpdateNotificationConfiguration,
	version types.ApplicationVersion,
	deployments []types.DeploymentPendingUpdate,
) error {
	log := internalctx.GetLogger(ctx).With(zap.Stringer("configId", config.ID))

	application := findApplication(config.Applications, version.ApplicationID)
	if application == nil {
		return fmt.Errorf("application %v is not linked to the configuration", version.ApplicationID)
	}

	organization, err := db.GetOrganizationWithBranding(ctx, config.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	var aggErr error
	for _, recipient := range config.Recipients {
		log := log.With(zap.Stringer("userId", recipient.ID))

		visible := visibleDeployments(deployments, recipient)
		if len(visible) == 0 {
			log.Debug("skip recipient without visible deployments")
			continue
		}

		if sent, err := db.NotificationRecordExists(ctx, recipient.ID, version.ID); err != nil {
			return fmt.Errorf("failed to check notification record: %w", err)
		} else if sent {
			log.Debug("skip recipient that was notified about this version already")
			continue
		}

		log.Info("sending update available notification")
		var message string
		if err := mailsending.ApplicationUpdateAvailableNotification(
			ctx, recipient, *organization, *application, version.Name, visible,
		); err != nil {
			log.Warn("update available notification sending failed", zap.Error(err))
			aggErr = errors.Join(aggErr, err)
			message = err.Error()
		}

		record := types.NotificationRecord{
			OrganizationID:         config.OrganizationID,
			CustomerOrganizationID: recipient.CustomerOrganizationID,
			UserAccountID:          &recipient.ID,
			SourceType:             types.NotificationSourceTypeApplication,
			SourceConfigurationID:  &config.ID,
			SubjectID:              &version.ID,
			Type:                   types.NotificationRecordTypeUpdateAvailable,
			Details: types.NotificationRecordDetails{
				Summary:                fmt.Sprintf("Version %v is available", version.Name),
				ApplicationName:        &application.Name,
				ApplicationType:        &application.Type,
				ApplicationVersionName: &version.Name,
				Deployments:            recordDeployments(visible),
			},
			Message: message,
		}
		if err := saveNotificationRecord(ctx, &record); err != nil {
			aggErr = errors.Join(aggErr, err)
		}
	}

	return aggErr
}

// SendArtifactVersionAvailableNotifications tells the recipients of every configuration that
// watches the version's artifact that the tag exists.
func SendArtifactVersionAvailableNotifications(ctx context.Context, version types.ArtifactVersion) error {
	// The row named after the manifest digest exists only to make the manifest pullable by digest,
	// and a multi-arch push creates one per child manifest. Only the tag is a release.
	if version.IsDigestVersion() {
		return nil
	}

	configs, err := db.GetUpdateNotificationConfigurationsForArtifact(ctx, version.ArtifactID)
	if err != nil {
		return fmt.Errorf("failed to get update notification configurations: %w", err)
	} else if len(configs) == 0 {
		return nil
	}

	entitlement, err := db.GetArtifactVersionEntitlement(ctx, version.ArtifactID, version.ID)
	if err != nil {
		return fmt.Errorf("failed to get artifact version entitlement: %w", err)
	}

	var aggErr error
	for _, config := range configs {
		if err := sendArtifactVersionAvailableWithConfig(ctx, config, version, entitlement); err != nil {
			aggErr = errors.Join(aggErr, fmt.Errorf("config %v: %w", config.ID, err))
		}
	}
	return aggErr
}

func sendArtifactVersionAvailableWithConfig(
	ctx context.Context,
	config types.UpdateNotificationConfiguration,
	version types.ArtifactVersion,
	entitlement types.ArtifactVersionEntitlement,
) error {
	log := internalctx.GetLogger(ctx).With(zap.Stringer("configId", config.ID))

	artifact := findArtifact(config.Artifacts, version.ArtifactID)
	if artifact == nil {
		return fmt.Errorf("artifact %v is not linked to the configuration", version.ArtifactID)
	}

	organization, err := db.GetOrganizationWithBranding(ctx, config.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	var aggErr error
	for _, recipient := range config.Recipients {
		log := log.With(zap.Stringer("userId", recipient.ID))

		if customerOrgID := recipient.CustomerOrganizationID; customerOrgID != nil &&
			!entitlement.Allows(*customerOrgID) {
			log.Debug("skip recipient whose entitlement does not cover the version")
			continue
		}

		if sent, err := db.NotificationRecordExists(ctx, recipient.ID, version.ID); err != nil {
			return fmt.Errorf("failed to check notification record: %w", err)
		} else if sent {
			log.Debug("skip recipient that was notified about this version already")
			continue
		}

		log.Info("sending new artifact version notification")
		var message string
		if err := mailsending.ArtifactVersionAvailableNotification(
			ctx, recipient, *organization, *artifact, version.Name,
		); err != nil {
			log.Warn("new artifact version notification sending failed", zap.Error(err))
			aggErr = errors.Join(aggErr, err)
			message = err.Error()
		}

		record := types.NotificationRecord{
			OrganizationID:         config.OrganizationID,
			CustomerOrganizationID: recipient.CustomerOrganizationID,
			UserAccountID:          &recipient.ID,
			SourceType:             types.NotificationSourceTypeArtifact,
			SourceConfigurationID:  &config.ID,
			SubjectID:              &version.ID,
			Type:                   types.NotificationRecordTypeNewVersion,
			Details: types.NotificationRecordDetails{
				Summary:             fmt.Sprintf("Version %v is available", version.Name),
				ArtifactName:        &artifact.Name,
				ArtifactVersionName: &version.Name,
			},
			Message: message,
		}
		if err := saveNotificationRecord(ctx, &record); err != nil {
			aggErr = errors.Join(aggErr, err)
		}
	}

	return aggErr
}

// deploymentsBehind narrows the deployments down to those that run a version the application
// orders before the announced one. A customer already on 2.1.0 has nothing to update to when a
// 2.0.5 bugfix is released. The ordering is the application's own, taken from all of its versions
// because the legacy strategy decides between SemVer and creation date for the whole set.
func deploymentsBehind(
	application types.Application,
	announced types.ApplicationVersion,
	deployments []types.DeploymentPendingUpdate,
) []types.DeploymentPendingUpdate {
	compare := types.ApplicationVersionComparator(application.VersioningStrategy, application.Versions)
	versions := make(map[uuid.UUID]types.ApplicationVersion, len(application.Versions))
	for _, version := range application.Versions {
		versions[version.ID] = version
	}

	behind := make([]types.DeploymentPendingUpdate, 0, len(deployments))
	for _, deployment := range deployments {
		if current, ok := versions[deployment.CurrentVersionID]; !ok || compare(current, announced) < 0 {
			behind = append(behind, deployment)
		}
	}
	return behind
}

// visibleDeployments narrows the affected deployments down to what a recipient may see. A customer
// user sees only their own organization's deployments and only while their entitlement covers the
// announced version; a partner user sees the deployments of the customers they manage, under the
// same entitlement rule; everyone else is a member of the vendor organization and sees all of them.
func visibleDeployments(
	deployments []types.DeploymentPendingUpdate,
	recipient types.NotificationRecipient,
) []types.DeploymentPendingUpdate {
	if recipient.CustomerOrganizationID == nil && recipient.PartnerOrganizationID == nil {
		return deployments
	}

	visible := make([]types.DeploymentPendingUpdate, 0, len(deployments))
	for _, deployment := range deployments {
		if !deployment.Entitled {
			continue
		}
		if !belongsTo(deployment.CustomerOrganizationID, recipient.CustomerOrganizationID) &&
			!belongsTo(deployment.PartnerOrganizationID, recipient.PartnerOrganizationID) {
			continue
		}
		visible = append(visible, deployment)
	}
	return visible
}

func belongsTo(deploymentOrgID, recipientOrgID *uuid.UUID) bool {
	return deploymentOrgID != nil && recipientOrgID != nil && *deploymentOrgID == *recipientOrgID
}

func recordDeployments(deployments []types.DeploymentPendingUpdate) []types.NotificationRecordDeployment {
	records := make([]types.NotificationRecordDeployment, 0, len(deployments))
	for _, deployment := range deployments {
		name := ""
		if deployment.ReleaseName != nil {
			name = *deployment.ReleaseName
		}
		records = append(records, types.NotificationRecordDeployment{
			CustomerOrganizationName: deployment.CustomerOrganizationName,
			DeploymentTargetName:     deployment.DeploymentTargetName,
			DeploymentName:           name,
			CurrentVersionName:       &deployment.CurrentVersionName,
		})
	}
	return records
}

// saveNotificationRecord tolerates a record another request wrote in the meantime: two pushes of
// the same tag racing each other must not fail the second send that the unique index rejects.
func saveNotificationRecord(ctx context.Context, record *types.NotificationRecord) error {
	if err := db.SaveNotificationRecord(ctx, record); err != nil {
		if errors.Is(err, db.ErrNotificationRecordExists) {
			internalctx.GetLogger(ctx).Debug("notification record was written concurrently")
			return nil
		}
		return fmt.Errorf("failed to save notification record: %w", err)
	}
	return nil
}

func findApplication(
	applications []types.NotificationApplication,
	id uuid.UUID,
) *types.NotificationApplication {
	for _, application := range applications {
		if application.ID == id {
			return &application
		}
	}
	return nil
}

func findArtifact(artifacts []types.NotificationArtifact, id uuid.UUID) *types.NotificationArtifact {
	for _, artifact := range artifacts {
		if artifact.ID == id {
			return &artifact
		}
	}
	return nil
}
