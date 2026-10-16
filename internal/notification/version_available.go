package notification

import (
	"context"
	"errors"
	"fmt"
	"slices"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mailsending"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

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
	mailed := map[uuid.UUID]struct{}{}
	for _, config := range configs {
		if err := sendApplicationUpdateAvailableWithConfig(ctx, config, version, deployments, mailed); err != nil {
			aggErr = errors.Join(aggErr, fmt.Errorf("config %v: %w", config.ID, err))
		}
	}
	return aggErr
}

// SendApplicationEntitlementVersionsNotifications announces the newest of the versions an
// entitlement was just widened to. A customer whose entitlement did not cover a version when it
// was created is skipped then, so this is the second and last moment at which they can learn about
// it. Recipients who were notified at creation time are held back by their notification records.
// No versions stands for all of them, as it does on the entitlement.
func SendApplicationEntitlementVersionsNotifications(
	ctx context.Context,
	organizationID, applicationID uuid.UUID,
	applicationVersionIDs []uuid.UUID,
) error {
	application, err := db.GetApplication(ctx, applicationID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get application: %w", err)
	}
	entitled := application.Versions
	if len(applicationVersionIDs) > 0 {
		entitled = slices.DeleteFunc(slices.Clone(entitled), func(version types.ApplicationVersion) bool {
			return !slices.Contains(applicationVersionIDs, version.ID)
		})
	}
	compare := types.ApplicationVersionComparator(application.VersioningStrategy, application.Versions)
	if version := types.LatestApplicationVersion(compare, entitled); version != nil {
		return SendApplicationUpdateAvailableNotifications(ctx, *version)
	}
	return nil
}

func sendApplicationUpdateAvailableWithConfig(
	ctx context.Context,
	config types.UpdateNotificationConfiguration,
	version types.ApplicationVersion,
	deployments []types.DeploymentPendingUpdate,
	mailed map[uuid.UUID]struct{},
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
	for _, group := range recipientsByAudience(config.Recipients) {
		log := group.audience.logger(log)

		visible := visibleDeployments(deployments, group.audience)
		if len(visible) == 0 {
			log.Debug("skip audience without visible deployments")
			continue
		}

		record := types.NotificationRecord{
			OrganizationID:                    config.OrganizationID,
			CustomerOrganizationID:            group.audience.customerOrganizationID,
			PartnerOrganizationID:             group.audience.partnerOrganizationID,
			UpdateNotificationConfigurationID: &config.ID,
			ApplicationVersionID:              &version.ID,
			Type:                              types.NotificationRecordTypeUpdateAvailable,
			Deployments:                       recordDeployments(visible),
		}
		if reserved, err := reserveNotificationRecord(ctx, &record); err != nil {
			return err
		} else if !reserved {
			log.Debug("skip audience that was notified about this version already")
			continue
		}

		log.Info("sending update available notification")
		var deliveryErr error
		for _, recipient := range unmailed(group.recipients, mailed) {
			if err := mailsending.ApplicationUpdateAvailableNotification(
				ctx, recipient, *organization, *application, version.Name, visible,
			); err != nil {
				log.Warn("update available notification sending failed",
					zap.Stringer("userId", recipient.ID), zap.Error(err))
				deliveryErr = errors.Join(deliveryErr, err)
			}
		}
		aggErr = errors.Join(aggErr, recordDeliveryError(ctx, record, deliveryErr))
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
	mailed := map[uuid.UUID]struct{}{}
	for _, config := range configs {
		if err := sendArtifactVersionAvailableWithConfig(ctx, config, version, entitlement, mailed); err != nil {
			aggErr = errors.Join(aggErr, fmt.Errorf("config %v: %w", config.ID, err))
		}
	}
	return aggErr
}

func SendArtifactEntitlementVersionsNotifications(
	ctx context.Context,
	artifactID uuid.UUID,
	artifactVersionIDs []uuid.UUID,
) error {
	version, err := db.GetNewestArtifactTag(ctx, artifactID, artifactVersionIDs)
	if err != nil {
		return fmt.Errorf("failed to get newest artifact tag: %w", err)
	} else if version == nil {
		return nil
	}
	return SendArtifactVersionAvailableNotifications(ctx, *version)
}

func sendArtifactVersionAvailableWithConfig(
	ctx context.Context,
	config types.UpdateNotificationConfiguration,
	version types.ArtifactVersion,
	entitlement types.GetArtifactVersionEntitlementResult,
	mailed map[uuid.UUID]struct{},
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
	for _, group := range recipientsByAudience(config.Recipients) {
		log := group.audience.logger(log)

		if customerOrgID := group.audience.customerOrganizationID; customerOrgID != nil &&
			!entitlement.Allows(*customerOrgID) {
			log.Debug("skip audience whose entitlement does not cover the version")
			continue
		}

		record := types.NotificationRecord{
			OrganizationID:                    config.OrganizationID,
			CustomerOrganizationID:            group.audience.customerOrganizationID,
			PartnerOrganizationID:             group.audience.partnerOrganizationID,
			UpdateNotificationConfigurationID: &config.ID,
			ArtifactVersionID:                 &version.ID,
			Type:                              types.NotificationRecordTypeUpdateAvailable,
		}
		if reserved, err := reserveNotificationRecord(ctx, &record); err != nil {
			return err
		} else if !reserved {
			log.Debug("skip audience that was notified about this version already")
			continue
		}

		log.Info("sending new artifact version notification")
		var deliveryErr error
		for _, recipient := range unmailed(group.recipients, mailed) {
			if err := mailsending.ArtifactVersionAvailableNotification(
				ctx, recipient, *organization, *artifact, version.Name,
			); err != nil {
				log.Warn("new artifact version notification sending failed",
					zap.Stringer("userId", recipient.ID), zap.Error(err))
				deliveryErr = errors.Join(deliveryErr, err)
			}
		}
		aggErr = errors.Join(aggErr, recordDeliveryError(ctx, record, deliveryErr))
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

// audience is who a recipient is notified as: the vendor's own team when both are nil, otherwise
// one partner or one customer. Everyone in an audience may see the same deployments, so an
// audience shares one notification record.
type audience struct {
	customerOrganizationID *uuid.UUID
	partnerOrganizationID  *uuid.UUID
}

func (a audience) logger(log *zap.Logger) *zap.Logger {
	if a.customerOrganizationID != nil {
		return log.With(zap.Stringer("customerOrganizationId", a.customerOrganizationID))
	} else if a.partnerOrganizationID != nil {
		return log.With(zap.Stringer("partnerOrganizationId", a.partnerOrganizationID))
	}
	return log
}

type audienceRecipients struct {
	audience   audience
	recipients []types.NotificationRecipient
}

func recipientsByAudience(recipients []types.NotificationRecipient) []audienceRecipients {
	var groups []audienceRecipients
	for _, recipient := range recipients {
		index := slices.IndexFunc(groups, func(group audienceRecipients) bool {
			return util.PtrEq(group.audience.customerOrganizationID, recipient.CustomerOrganizationID) &&
				util.PtrEq(group.audience.partnerOrganizationID, recipient.PartnerOrganizationID)
		})
		if index < 0 {
			groups = append(groups, audienceRecipients{audience: audience{
				customerOrganizationID: recipient.CustomerOrganizationID,
				partnerOrganizationID:  recipient.PartnerOrganizationID,
			}})
			index = len(groups) - 1
		}
		groups[index].recipients = append(groups[index].recipients, recipient)
	}
	return groups
}

// unmailed drops the recipients another configuration has mailed about the same version already
// and marks the rest as mailed, so a user listed in several configurations receives one email.
func unmailed(recipients []types.NotificationRecipient, mailed map[uuid.UUID]struct{}) []types.NotificationRecipient {
	result := make([]types.NotificationRecipient, 0, len(recipients))
	for _, recipient := range recipients {
		if _, ok := mailed[recipient.ID]; !ok {
			mailed[recipient.ID] = struct{}{}
			result = append(result, recipient)
		}
	}
	return result
}

// visibleDeployments narrows the affected deployments down to what an audience may see. A customer
// sees only its own deployments, a partner sees the deployments of the customers it manages and
// the vendor's own team sees all of them.
func visibleDeployments(
	deployments []types.DeploymentPendingUpdate,
	audience audience,
) []types.DeploymentPendingUpdate {
	if audience.customerOrganizationID == nil && audience.partnerOrganizationID == nil {
		return deployments
	}

	visible := make([]types.DeploymentPendingUpdate, 0, len(deployments))
	for _, deployment := range deployments {
		if !belongsTo(deployment.CustomerOrganizationID, audience.customerOrganizationID) &&
			!belongsTo(deployment.PartnerOrganizationID, audience.partnerOrganizationID) {
			continue
		}
		visible = append(visible, deployment)
	}
	return visible
}

func belongsTo(deploymentOrgID, audienceOrgID *uuid.UUID) bool {
	return deploymentOrgID != nil && audienceOrgID != nil && *deploymentOrgID == *audienceOrgID
}

func recordDeployments(deployments []types.DeploymentPendingUpdate) []types.NotificationRecordDeployment {
	records := make([]types.NotificationRecordDeployment, 0, len(deployments))
	for _, deployment := range deployments {
		records = append(records, types.NotificationRecordDeployment{
			CustomerOrganizationName: deployment.CustomerOrganizationName,
			DeploymentTargetName:     deployment.DeploymentTargetName,
			HelmReleaseName:          deployment.HelmReleaseName,
			CurrentVersionName:       deployment.CurrentVersionName,
		})
	}
	return records
}

// reserveNotificationRecord writes the record before the mails are sent, so that the unique index
// lets only one of two racing sends through to the same audience. It reports false when another
// send holds the record already.
func reserveNotificationRecord(ctx context.Context, record *types.NotificationRecord) (bool, error) {
	if err := db.SaveNotificationRecord(ctx, record); errors.Is(err, db.ErrNotificationRecordExists) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to save notification record: %w", err)
	}
	return true, nil
}

// recordDeliveryError returns deliveryErr together with any error of recording it.
func recordDeliveryError(ctx context.Context, record types.NotificationRecord, deliveryErr error) error {
	if deliveryErr == nil {
		return nil
	}
	return errors.Join(deliveryErr, db.SetNotificationRecordDeliveryError(ctx, record.ID, deliveryErr.Error()))
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
