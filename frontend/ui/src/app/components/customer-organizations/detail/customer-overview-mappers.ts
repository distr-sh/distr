import {Application, DeploymentTarget} from '@distr-sh/distr-sdk';
import dayjs, {Dayjs} from 'dayjs';
import {isDeploymentOutdated} from '../../../../util/versions';
import {
  DeploymentDisplayStatus,
  deploymentDisplayStatus,
} from '../../../deployments/deployment-target-card/deployment-status-text.component';
import {licenseExpiryWindowDays} from '../../../home/vendor/vendor-home-mappers';
import {ApplicationEntitlement} from '../../../types/application-entitlement';
import {LatestArtifactPull} from '../../../types/artifact-version-pull';
import {License} from '../../../types/license';
import {CustomerLicenseStatus, toCustomerLicenseStatus} from '../customer-list-mappers';

export interface CustomerOverviewUpdates {
  /** What `outdated` counts: the deployments, or the downloaded artifacts of a customer without deployments. */
  source: 'deployments' | 'downloads' | 'none';
  outdated: number;
}

export interface CustomerOverviewLicense {
  status: CustomerLicenseStatus;
  /** Whether any customer has a license, which is what makes a missing one an issue. */
  isLicensingInUse: boolean;
  grants: number;
  expired: number;
  expiring: number;
}

export function toAffectedTargetNames(targets: DeploymentTarget[], status: DeploymentDisplayStatus): string[] {
  return targets
    .filter((target) => target.deployments.some((it) => deploymentDisplayStatus(it) === status))
    .map((target) => target.name);
}

/** Judges the customer by its deployments and by its artifact pulls only without any, like `toCustomerUpdates`. */
export function toCustomerOverviewUpdates(
  targets: DeploymentTarget[],
  applications: Application[],
  entitlements: ApplicationEntitlement[],
  pulls: LatestArtifactPull[]
): CustomerOverviewUpdates {
  const deployments = targets.flatMap((target) => target.deployments);
  if (deployments.length > 0) {
    const outdated = deployments.filter((it) => isDeploymentOutdated(it, applications, entitlements)).length;
    return {source: 'deployments', outdated};
  } else if (pulls.length > 0) {
    return {source: 'downloads', outdated: pulls.filter((pull) => pull.availableUpdate !== undefined).length};
  } else {
    return {source: 'none', outdated: 0};
  }
}

/** Counts the license keys and entitlements of the customer that have expired or expire within the window. */
export function toCustomerOverviewLicense(
  customerOrganizationId: string,
  licenses: License[],
  now: Dayjs = dayjs()
): CustomerOverviewLicense {
  const grants = (license: License) => [
    ...license.licenseKeys,
    ...license.applicationEntitlements,
    ...license.artifactEntitlements,
  ];
  const license = licenses.find((it) => it.customerOrganization.id === customerOrganizationId);
  const customerGrants = license ? grants(license) : [];
  const expirations = customerGrants.map((it) => it.expiresAt).filter((expiresAt) => expiresAt !== undefined);
  const threshold = now.add(licenseExpiryWindowDays, 'day');
  const expired = expirations.filter((expiresAt) => !now.isBefore(expiresAt)).length;
  return {
    status: toCustomerLicenseStatus(license, now),
    isLicensingInUse: licenses.some((it) => grants(it).length > 0),
    grants: customerGrants.length,
    expired,
    expiring: expirations.filter((expiresAt) => !threshold.isBefore(expiresAt)).length - expired,
  };
}
