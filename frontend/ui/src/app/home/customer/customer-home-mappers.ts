import {Application, DeploymentTarget, UserAccount} from '@distr-sh/distr-sdk';
import dayjs, {Dayjs} from 'dayjs';
import {latestDeploymentVersion} from '../../../util/versions';
import {Artifact} from '../../services/artifacts.service';
import {AccessToken, nextAccessTokenExpiry} from '../../types/access-token';
import {ActivityFeedEntry} from '../../types/activity-feed';
import {AlertConfiguration} from '../../types/alert-configuration';
import {ApplicationEntitlement} from '../../types/application-entitlement';
import {ArtifactEntitlement} from '../../types/artifact-entitlement';
import {
  CustomerHomeAccessToken,
  CustomerHomeActivity,
  CustomerHomeAlerts,
  CustomerHomeArtifacts,
  CustomerHomeChecklist,
  CustomerHomeDeploymentTarget,
  CustomerHomeLicense,
  CustomerHomeSupportBundle,
  CustomerHomeUser,
} from '../../types/customer-home';
import {LicenseKey} from '../../types/license-key';
import {UpdateNotificationConfiguration} from '../../types/notification-configuration';
import {NotificationRecord} from '../../types/notification-record';
import {SupportBundle} from '../../types/support-bundle';

const accessTokenExpiryWindowDays = 7;
const notificationWindowDays = 7;

function user(name: string | undefined, imageUrl: string | undefined): CustomerHomeUser | undefined {
  return name ? {name, imageUrl} : undefined;
}

export function toCustomerHomeDeploymentTargets(
  targets: DeploymentTarget[],
  applications: Application[],
  entitlements: ApplicationEntitlement[]
): CustomerHomeDeploymentTarget[] {
  return targets.map((target) => ({
    id: target.id!,
    name: target.name,
    type: target.type,
    connected: target.reportedControllerVersionId !== undefined,
    deployments: target.deployments.map((deployment) => {
      const application = applications.find((app) => app.id === deployment.application.id);
      const latest = latestDeploymentVersion(deployment, applications, entitlements);
      return {
        id: deployment.id!,
        applicationId: deployment.application.id!,
        applicationName: deployment.application.name ?? '',
        applicationImageUrl: application?.imageUrl ?? deployment.application.imageUrl,
        applicationTitleImageUrl: application?.titleImageUrl,
        applicationLink: deployment.applicationLink || undefined,
        applicationVersionId: deployment.applicationVersionId,
        applicationVersionName: deployment.applicationVersionName,
        currentApplicationVersionId: deployment.currentApplicationVersionId,
        currentApplicationVersionName: deployment.currentApplicationVersionName,
        latestApplicationVersionName: latest?.name,
        updateAvailable:
          latest !== undefined &&
          latest.id !== (deployment.currentApplicationVersionId ?? deployment.applicationVersionId),
        deploymentRevisionId: deployment.deploymentRevisionId,
        deploymentRevisionCreatedAt: deployment.deploymentRevisionCreatedAt,
        currentDeploymentRevisionId: deployment.currentDeploymentRevisionId,
        latestStatus: deployment.latestStatus,
        currentStatus: deployment.currentStatus,
      };
    }),
  }));
}

export function toCustomerHomeArtifacts(artifacts: Artifact[]): CustomerHomeArtifacts {
  return {
    hasPulled: artifacts.some((artifact) => artifact.lastPull !== undefined),
    artifacts: artifacts
      .filter((artifact) => artifact.featured)
      .map((artifact) => ({
        id: artifact.id,
        name: artifact.name,
        imageUrl: artifact.imageUrl,
        titleImageUrl: artifact.titleImageUrl,
        lastPull: artifact.lastPull && {
          versionName: artifact.lastPull.versionName,
          createdAt: artifact.lastPull.createdAt,
          pulledBy: user(artifact.lastPull.pulledByName, artifact.lastPull.pulledByImageUrl),
          deploymentTargetName: artifact.lastPull.deploymentTargetName,
        },
      })),
  };
}

export interface CustomerHomeChecklistSources {
  user: UserAccount;
  userCount: number;
  deploymentTargets: DeploymentTarget[];
  secretCount: number;
  accessTokenCount: number;
  artifacts: Artifact[];
  /** Undefined when the customer has no update notifications. */
  updateNotificationConfigurations: UpdateNotificationConfiguration[] | undefined;
}

export function toCustomerHomeChecklist(sources: CustomerHomeChecklistSources): CustomerHomeChecklist {
  return {
    profileCompleted: sources.user.imageId !== undefined || sources.user.mfaEnabled,
    teamMembersInvited: sources.userCount >= 2,
    deploymentCreated: sources.deploymentTargets.some((target) => target.deployments.length > 0),
    secretCreated: sources.secretCount > 0,
    accessTokenCreated: sources.accessTokenCount > 0,
    artifactPulled: sources.artifacts.some((artifact) => artifact.lastPull !== undefined),
    updateNotificationsConfigured:
      sources.updateNotificationConfigurations && sources.updateNotificationConfigurations.length > 0,
  };
}

/**
 * What decides the license state of the customer. License keys replace each other, so only the best one
 * counts: a valid key before one that is not valid yet before an expired one and among those the key that
 * expires last. Entitlements cover different applications and artifacts, so each one counts. Of that key
 * and the entitlements, the most urgent decides: an expired one before a key that is not valid yet before
 * the one that expires first.
 */
export function toCustomerHomeLicense(
  licenseKeys: LicenseKey[],
  entitlements: (ApplicationEntitlement | ArtifactEntitlement)[],
  now: Dayjs = dayjs()
): CustomerHomeLicense | undefined {
  const expired = ({expiresAt}: CustomerHomeLicense) => expiresAt !== undefined && !now.isBefore(expiresAt);
  const notYetValid = ({notBefore}: CustomerHomeLicense) => notBefore !== undefined && now.isBefore(notBefore);
  const expiry = ({expiresAt}: CustomerHomeLicense) =>
    expiresAt === undefined ? Number.MAX_SAFE_INTEGER : dayjs(expiresAt).valueOf();
  const byName = (a: CustomerHomeLicense, b: CustomerHomeLicense) => a.name.localeCompare(b.name);

  const hasLicenseKeys = licenseKeys.length > 0;
  const keyRank = (key: CustomerHomeLicense) => (expired(key) ? 2 : notYetValid(key) ? 1 : 0);
  const licenseKey = licenseKeys
    .map((key) => ({name: key.name ?? '', notBefore: key.notBefore, expiresAt: key.expiresAt, hasLicenseKeys}))
    .sort((a, b) => keyRank(a) - keyRank(b) || expiry(b) - expiry(a) || byName(a, b))
    .at(0);

  const urgency = (license: CustomerHomeLicense) => (expired(license) ? 0 : notYetValid(license) ? 1 : 2);
  return [
    ...(licenseKey ? [licenseKey] : []),
    ...entitlements.map((entitlement) => ({
      name: entitlement.name ?? '',
      expiresAt: entitlement.expiresAt && dayjs(entitlement.expiresAt).toISOString(),
      hasLicenseKeys,
    })),
  ]
    .sort((a, b) => urgency(a) - urgency(b) || expiry(a) - expiry(b) || byName(a, b))
    .at(0);
}

/** The start of the window the alerts panel counts notifications in. */
export function customerHomeNotificationsSince(now: Dayjs = dayjs()): Date {
  return now.subtract(notificationWindowDays, 'day').toDate();
}

/** `records` have to be those since customerHomeNotificationsSince. */
export function toCustomerHomeAlerts(
  configurations: AlertConfiguration[],
  records: NotificationRecord[]
): CustomerHomeAlerts {
  return {
    alertConfigurationCount: configurations.length,
    notificationCount: records.filter((record) => record.type === 'alert' || record.type === 'warning').length,
  };
}
/** The tokens that have expired or expire within the next days, the soonest first. */
export function toCustomerHomeAccessTokens(tokens: AccessToken[], now: Dayjs = dayjs()): CustomerHomeAccessToken[] {
  const threshold = now.add(accessTokenExpiryWindowDays, 'day');
  return tokens
    .map((token) => ({token, expiresAt: nextAccessTokenExpiry(token, now)}))
    .filter(({expiresAt}) => expiresAt !== undefined && dayjs(expiresAt).isBefore(threshold))
    .sort((a, b) => dayjs(a.expiresAt).valueOf() - dayjs(b.expiresAt).valueOf())
    .map(({token, expiresAt}) => ({
      id: token.id!,
      label: token.label,
      expiresAt: expiresAt!,
      lastUsedAt: token.lastUsedAt,
    }));
}

export function toCustomerHomeSupportBundles(bundles: SupportBundle[]): CustomerHomeSupportBundle[] {
  return bundles
    .map((bundle) => ({
      id: bundle.id,
      title: bundle.title,
      status: bundle.status,
      updatedAt: bundle.statusChangedAt ?? bundle.createdAt,
      // Only the vendor resolves a bundle, and the customer must not see who of the vendor did.
      updatedBy:
        bundle.status === 'resolved'
          ? undefined
          : bundle.statusChangedAt
            ? user(bundle.statusChangedByUserName, bundle.statusChangedByImageUrl)
            : user(bundle.createdByUserName, bundle.createdByImageUrl),
    }))
    .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
}

export function toCustomerHomeActivity(entries: ActivityFeedEntry[]): CustomerHomeActivity[] {
  return entries.map((entry) => ({
    id: entry.id,
    createdAt: entry.createdAt,
    type: entry.type,
    resourceId: entry.resourceId,
    subject: entry.subject,
    detail: entry.detail,
    user: user(entry.userName, entry.userImageUrl),
  }));
}
