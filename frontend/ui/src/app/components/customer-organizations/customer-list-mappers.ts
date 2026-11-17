import {Application, CustomerOrganizationWithUsage, DeploymentTarget} from '@distr-sh/distr-sdk';
import dayjs, {Dayjs} from 'dayjs';
import {DeploymentHealth} from '../../home/vendor/vendor-home';
import {
  filterOpenSupportBundles,
  licenseExpiryWindowDays,
  toCustomerUpdates,
  toDeploymentHealth,
} from '../../home/vendor/vendor-home-mappers';
import {ApplicationEntitlement} from '../../types/application-entitlement';
import {LatestArtifactPull} from '../../types/artifact-version-pull';
import {License} from '../../types/license';
import {SupportBundle, SupportBundleStatus} from '../../types/support-bundle';
import {CustomerDetailTabId} from './detail/customer-detail-navigation';

export type CustomerManagement = 'controller' | 'self' | 'unknown';

export type CustomerLicenseStatus = 'no_license' | 'expired' | 'expiring' | 'valid';

export interface CustomerListRow {
  id: string;
  name: string;
  imageUrl?: string;
  /** Undefined while the deployments and pulls of a customer with both features are loading. */
  management?: CustomerManagement;
  /** The tab that shows what the updates column judges: the deployments, or the downloads of a customer without. */
  updatesTab: Extract<CustomerDetailTabId, 'deployments' | 'downloads'>;
  /** Undefined while the licenses are loading or the organization has no licensing. */
  license?: CustomerLicenseStatus;
  /** Undefined while the deployments and pulls are loading. */
  deployments?: CustomerListDeployments;
  /** Undefined while the deployments and pulls are loading. */
  isOutdated?: boolean;
  /** Undefined while the support bundles are loading or the organization's plan has none. */
  supportBundles?: CustomerListSupportBundles;
}

export interface CustomerListDeployments extends DeploymentHealth {
  noStatus: number;
}

export interface CustomerListSupportBundles {
  open: number;
  resolved: number;
  canceled: number;
}

export interface CustomerListDeploymentSources {
  targets: DeploymentTarget[];
  applications: Application[];
  entitlements: ApplicationEntitlement[];
  pulls: LatestArtifactPull[];
}

/**
 * Decided by the features when the customer has only one of deployment targets and artifacts, and by whether it
 * deployed or pulled anything when it has both, since every customer gets both by default.
 */
function toCustomerManagement(
  customer: CustomerOrganizationWithUsage,
  health: DeploymentHealth | undefined,
  pulls: LatestArtifactPull[] | undefined
): CustomerManagement | undefined {
  const deploymentTargetsEnabled = customer.features.includes('deployment_targets');
  const artifactsEnabled = customer.features.includes('artifacts');
  if (deploymentTargetsEnabled && !artifactsEnabled) {
    return 'controller';
  } else if (artifactsEnabled && !deploymentTargetsEnabled) {
    return 'self';
  } else if (!deploymentTargetsEnabled) {
    return 'unknown';
  } else if (health === undefined || pulls === undefined) {
    return undefined;
  } else if (health.total > 0) {
    return 'controller';
  } else if (pulls.some((pull) => pull.customerOrganizationId === customer.id)) {
    return 'self';
  } else {
    return 'unknown';
  }
}

export function toCustomerLicenseStatus(license: License | undefined, now: Dayjs = dayjs()): CustomerLicenseStatus {
  const grants = license
    ? [...license.licenseKeys, ...license.applicationEntitlements, ...license.artifactEntitlements]
    : [];
  const expirations = grants.map((it) => it.expiresAt).filter((expiresAt) => expiresAt !== undefined);
  const threshold = now.add(licenseExpiryWindowDays, 'day');
  if (grants.length === 0) {
    return 'no_license';
  } else if (expirations.some((expiresAt) => !now.isBefore(expiresAt))) {
    return 'expired';
  } else if (expirations.some((expiresAt) => !threshold.isBefore(expiresAt))) {
    return 'expiring';
  } else {
    return 'valid';
  }
}

export function toCustomerListRows(
  customers: CustomerOrganizationWithUsage[],
  deployments: CustomerListDeploymentSources | undefined,
  licenses: License[] | undefined,
  supportBundles: SupportBundle[] | undefined,
  now: Dayjs = dayjs()
): CustomerListRow[] {
  const licenseByCustomer = new Map(licenses?.map((license) => [license.customerOrganization.id, license]));
  return customers.map((customer): CustomerListRow => {
    const health =
      deployments &&
      toDeploymentHealth(deployments.targets.filter((target) => target.customerOrganization?.id === customer.id));
    return {
      id: customer.id,
      name: customer.name,
      imageUrl: customer.imageUrl,
      management: toCustomerManagement(customer, health, deployments?.pulls),
      updatesTab: (health?.total ?? 0) > 0 ? 'deployments' : 'downloads',
      license: licenses && toCustomerLicenseStatus(licenseByCustomer.get(customer.id), now),
      deployments: health && {
        ...health,
        noStatus: health.total - health.healthy - health.running - health.error - health.stale - health.progressing,
      },
      isOutdated:
        deployments &&
        toCustomerUpdates(
          customer,
          deployments.targets,
          deployments.applications,
          deployments.entitlements,
          deployments.pulls
        ).isOutdated,
      supportBundles:
        supportBundles &&
        toCustomerSupportBundles(supportBundles.filter((bundle) => bundle.customerOrganizationId === customer.id)),
    };
  });
}

function toCustomerSupportBundles(bundles: SupportBundle[]): CustomerListSupportBundles {
  const count = (status: SupportBundleStatus) => bundles.filter((bundle) => bundle.status === status).length;
  return {
    open: filterOpenSupportBundles(bundles).length,
    resolved: count('resolved'),
    canceled: count('canceled'),
  };
}
