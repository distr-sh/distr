import {
  Advisory,
  AdvisoryStatus,
  Application,
  CustomerOrganization,
  CustomerOrganizationWithUsage,
  DeploymentTarget,
} from '@distr-sh/distr-sdk';
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

export function filterOpenSupportBundles(bundles: SupportBundle[]): SupportBundle[] {
  return bundles.filter((bundle) => openSupportBundleStatuses.includes(bundle.status));
}

/** The advisory statuses the advisories card counts and links to. */
export const vendorHomeOpenAdvisoryStatuses: AdvisoryStatus[] = ['triage', 'draft', 'published'];

export function toVendorHomeAdvisoryCounts(advisories: Advisory[]): VendorHomeAdvisoryCounts {
  const count = (status: AdvisoryStatus) => advisories.reduce((c, it) => (it.status === status ? c + 1 : c), 0);
  return {
    total: advisories.length,
    open: advisories.filter((it) => vendorHomeOpenAdvisoryStatuses.includes(it.status)).length,
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

function customerTargets(customer: CustomerOrganization, targets: DeploymentTarget[]): DeploymentTarget[] {
  return targets.filter((target) => target.customerOrganization?.id === customer.id);
}

/**
 * A customer is outdated when one of its deployments is. A customer without deployments is outdated when it
 * pulled an artifact it may pull a newer tag of.
 */
function toCustomerUpdates(
  customer: CustomerOrganization,
  targets: DeploymentTarget[],
  applications: Application[],
  entitlements: ApplicationEntitlement[],
  pulls: LatestArtifactPull[]
): {isOutdated: boolean; outdatedDeployments: number} {
  const deployments = customerTargets(customer, targets).flatMap((target) => target.deployments);
  const outdatedDeployments = deployments.filter((it) => isDeploymentOutdated(it, applications, entitlements)).length;
  const isOutdated =
    deployments.length > 0
      ? outdatedDeployments > 0
      : pulls.some((pull) => pull.customerOrganizationId === customer.id && pull.availableUpdate !== undefined);
  return {isOutdated, outdatedDeployments};
}

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
    const updates = toCustomerUpdates(customer, targets, applications, entitlements, pulls);
    if (updates.isOutdated) {
      outdatedCustomers++;
    }
    outdatedDeployments += updates.outdatedDeployments;
  }
  return {totalCustomers: customers.length, outdatedCustomers, outdatedDeployments};
}

function toCustomerUsage(
  customer: CustomerOrganizationWithUsage,
  deployments: number,
  pulls: LatestArtifactPull[]
): VendorHomeCustomer['usage'] {
  if (customer.features.includes('deployment_targets')) {
    return {type: 'deployments', count: deployments};
  } else if (customer.features.includes('artifacts')) {
    const artifactIds = new Set(
      pulls.filter((pull) => pull.customerOrganizationId === customer.id).map((pull) => pull.artifactId)
    );
    return {type: 'downloads', count: artifactIds.size};
  } else {
    return {type: 'users', count: customer.userCount};
  }
}

/**
 * Counts as open items of a customer every deployment in error or stale, every outdated deployment (or the outdated
 * artifact pulls of a customer without deployments, as one item), an expired or expiring license and every open
 * support request. Customers with open items come first, the most first, and the rest follow alphabetically.
 * `licenses` and `openSupportBundles` are `null` while the organization lacks the feature.
 */
export function toVendorHomeCustomers(
  customers: CustomerOrganizationWithUsage[],
  targets: DeploymentTarget[],
  applications: Application[],
  entitlements: ApplicationEntitlement[],
  pulls: LatestArtifactPull[],
  licenses: VendorHomeLicenses | null,
  openSupportBundles: SupportBundle[] | null
): VendorHomeCustomer[] {
  const licenseCustomerIds = new Set([
    ...(licenses?.expiredCustomerIds ?? []),
    ...(licenses?.expiringCustomerIds ?? []),
  ]);
  return customers
    .map((customer) => {
      const health = toVendorHomeDeploymentHealth(customerTargets(customer, targets));
      const updates = toCustomerUpdates(customer, targets, applications, entitlements, pulls);
      const openItems =
        health.error +
        health.stale +
        (updates.outdatedDeployments > 0 ? updates.outdatedDeployments : updates.isOutdated ? 1 : 0) +
        (licenseCustomerIds.has(customer.id) ? 1 : 0) +
        (openSupportBundles ?? []).filter((bundle) => bundle.customerOrganizationId === customer.id).length;
      return {
        id: customer.id,
        name: customer.name,
        imageUrl: customer.imageUrl,
        usage: toCustomerUsage(customer, health.total, pulls),
        openItems,
      };
    })
    .sort((a, b) => b.openItems - a.openItems || a.name.localeCompare(b.name));
}
