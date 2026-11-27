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
import {SupportBundle} from '../../types/support-bundle';
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
  updatesTab: Extract<CustomerDetailTabId, 'deployments' | 'registry'>;
  /** Undefined while the licenses are loading or the organization has no licensing. */
  license?: CustomerLicenseStatus;
  /** Undefined while the licenses are loading or the organization has no licensing. */
  isLicensingInUse?: boolean;
  /** Undefined while the deployments and pulls are loading. */
  deployments?: CustomerListDeployments;
  /** Undefined while the deployments and pulls are loading. */
  isOutdated?: boolean;
  /** Undefined while the deployments and pulls are loading. */
  hasDeploymentsOrPulls?: boolean;
  /** Undefined while the support bundles are loading or the organization's plan has none. */
  supportBundles?: CustomerListSupportBundles;
}

export interface CustomerListDeployments extends DeploymentHealth {
  noStatus: number;
}

export interface CustomerListSupportBundles {
  open: number;
  resolved: number;
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
  } else if (pulls.length > 0) {
    return 'self';
  } else {
    return 'unknown';
  }
}

export interface LicenseGrantCounts {
  grants: number;
  expired: number;
  /** The grants that expire within the window but have not expired yet. */
  expiring: number;
}

export function toLicenseGrantCounts(license: License | undefined, now: Dayjs = dayjs()): LicenseGrantCounts {
  const grants = license
    ? [...license.licenseKeys, ...license.applicationEntitlements, ...license.artifactEntitlements]
    : [];
  const expirations = grants.map((it) => it.expiresAt).filter((expiresAt) => expiresAt !== undefined);
  const threshold = now.add(licenseExpiryWindowDays, 'day');
  const expired = expirations.filter((expiresAt) => !now.isBefore(expiresAt)).length;
  return {
    grants: grants.length,
    expired,
    expiring: expirations.filter((expiresAt) => !threshold.isBefore(expiresAt)).length - expired,
  };
}

export function toCustomerLicenseStatus(license: License | undefined, now: Dayjs = dayjs()): CustomerLicenseStatus {
  const counts = toLicenseGrantCounts(license, now);
  if (counts.grants === 0) {
    return 'no_license';
  } else if (counts.expired > 0) {
    return 'expired';
  } else if (counts.expiring > 0) {
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
  const licenseStatuses = new Map<string, CustomerLicenseStatus>(
    licenses
      ? customers.map((customer) => [customer.id, toCustomerLicenseStatus(licenseByCustomer.get(customer.id), now)])
      : []
  );
  const isLicensingInUse = [...licenseStatuses.values()].some((status) => status !== 'no_license');
  return customers
    .map((customer): CustomerListRow => {
      const health =
        deployments &&
        toDeploymentHealth(deployments.targets.filter((target) => target.customerOrganization?.id === customer.id));
      const pulls = deployments?.pulls.filter((pull) => pull.customerOrganizationId === customer.id);
      const updates =
        deployments &&
        toCustomerUpdates(
          customer,
          deployments.targets,
          deployments.applications,
          deployments.entitlements,
          deployments.pulls
        );
      return {
        id: customer.id,
        name: customer.name,
        imageUrl: customer.imageUrl,
        management: toCustomerManagement(customer, health, pulls),
        updatesTab: (health?.total ?? 0) > 0 ? 'deployments' : 'registry',
        license: licenseStatuses.get(customer.id),
        isLicensingInUse: licenses && isLicensingInUse,
        deployments: health && {
          ...health,
          noStatus: health.total - health.healthy - health.running - health.error - health.stale - health.progressing,
        },
        isOutdated: updates?.isOutdated,
        hasDeploymentsOrPulls: updates?.hasDeploymentsOrPulls,
        supportBundles:
          supportBundles &&
          toCustomerSupportBundles(supportBundles.filter((bundle) => bundle.customerOrganizationId === customer.id)),
      };
    })
    .sort((a, b) => a.name.localeCompare(b.name));
}

function toCustomerSupportBundles(bundles: SupportBundle[]): CustomerListSupportBundles {
  return {
    open: filterOpenSupportBundles(bundles).length,
    resolved: bundles.filter((bundle) => bundle.status === 'resolved').length,
  };
}
