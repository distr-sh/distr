import {Advisory, AdvisoryStatus, Application, CustomerOrganization, DeploymentTarget} from '@distr-sh/distr-sdk';
import dayjs, {Dayjs} from 'dayjs';
import {isDeploymentOutdated} from '../../../util/versions';
import {
  DeploymentDisplayStatus,
  deploymentDisplayStatus,
} from '../../deployments/deployment-target-card/deployment-status-text.component';
import {ApplicationEntitlement} from '../../types/application-entitlement';
import {LatestArtifactPull} from '../../types/artifact-version-pull';
import {License} from '../../types/license';
import {SupportBundle, SupportBundleStatus} from '../../types/support-bundle';
import {
  VendorHomeAdvisoryCounts,
  VendorHomeCustomer,
  VendorHomeDeploymentHealth,
  VendorHomeLicenses,
  VendorHomeUpdates,
} from './vendor-home';

/** The status filter the deployments card links to, which has to show exactly the error and stale deployments. */
export const vendorHomeAttentionStatuses: readonly DeploymentDisplayStatus[] = ['error', 'stale'];

export const vendorHomeLicenseExpiryWindowDays = 30;

const openSupportBundleStatuses: readonly SupportBundleStatus[] = ['initialized', 'created'];

/**
 * The customers with an expired license key or entitlement, and those without one but with one that expires within
 * the window, so that every customer counts once.
 */
export function toVendorHomeLicenses(licenses: License[], now: Dayjs = dayjs()): VendorHomeLicenses {
  const threshold = now.add(vendorHomeLicenseExpiryWindowDays, 'day');
  const expirations = (license: License) =>
    [...license.licenseKeys, ...license.applicationEntitlements, ...license.artifactEntitlements]
      .map((it) => it.expiresAt)
      .filter((expiresAt) => expiresAt !== undefined);
  const result: VendorHomeLicenses = {expiredCustomerIds: [], expiringCustomerIds: []};
  for (const license of licenses) {
    const dates = expirations(license);
    if (dates.some((expiresAt) => !now.isBefore(expiresAt))) {
      result.expiredCustomerIds.push(license.customerOrganization.id);
    } else if (dates.some((expiresAt) => !threshold.isBefore(expiresAt))) {
      result.expiringCustomerIds.push(license.customerOrganization.id);
    }
  }
  return result;
}

export function countOpenSupportBundles(bundles: SupportBundle[]): number {
  return bundles.filter((bundle) => openSupportBundleStatuses.includes(bundle.status)).length;
}

/** The advisory statuses the advisories card counts and links to. */
export const vendorHomeOpenAdvisoryStatuses: AdvisoryStatus[] = ['triage', 'draft', 'published'];

export function toVendorHomeAdvisoryCounts(advisories: Advisory[]): VendorHomeAdvisoryCounts {
  const count = (status: AdvisoryStatus) => advisories.reduce((c, it) => (it.status === status ? c + 1 : c), 0);
  return {
    total: advisories.length,
    triage: count('triage'),
    draft: count('draft'),
    published: count('published'),
  };
}

export function toVendorHomeDeploymentHealth(targets: DeploymentTarget[]): VendorHomeDeploymentHealth {
  const statuses = targets.flatMap((target) => target.deployments.map(deploymentDisplayStatus));
  const count = (status: DeploymentDisplayStatus) => statuses.reduce((c, it) => (it === status ? c + 1 : c), 0);
  return {
    total: statuses.length,
    healthy: count('healthy'),
    running: count('running'),
    error: count('error'),
    stale: count('stale'),
  };
}

/**
 * A customer is outdated when one of its deployments is. A customer without deployments is outdated when it
 * pulled an artifact it may pull a newer tag of.
 */
export function toVendorHomeUpdates(
  customers: CustomerOrganization[],
  targets: DeploymentTarget[],
  applications: Application[],
  entitlements: ApplicationEntitlement[],
  pulls: LatestArtifactPull[]
): VendorHomeUpdates {
  let outdatedCustomers = 0;
  let outdatedDeployments = 0;
  for (const customer of customers) {
    const deployments = targets
      .filter((target) => target.customerOrganization?.id === customer.id)
      .flatMap((target) => target.deployments);
    const outdated = deployments.filter((it) => isDeploymentOutdated(it, applications, entitlements)).length;
    const isOutdated =
      deployments.length > 0
        ? outdated > 0
        : pulls.some((pull) => pull.customerOrganizationId === customer.id && pull.availableUpdate !== undefined);
    if (isOutdated) {
      outdatedCustomers++;
    }
    outdatedDeployments += outdated;
  }
  return {totalCustomers: customers.length, outdatedCustomers, outdatedDeployments};
}

export function toVendorHomeCustomers(
  customers: CustomerOrganization[],
  targets: DeploymentTarget[]
): VendorHomeCustomer[] {
  return customers.map((customer) => ({
    id: customer.id,
    name: customer.name,
    imageUrl: customer.imageUrl,
    deploymentHealth: toVendorHomeDeploymentHealth(
      targets.filter((target) => target.customerOrganization?.id === customer.id)
    ),
  }));
}
