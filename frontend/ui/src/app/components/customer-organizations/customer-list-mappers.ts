import {Application, CustomerOrganizationWithUsage, DeploymentTarget} from '@distr-sh/distr-sdk';
import dayjs, {Dayjs} from 'dayjs';
import {VendorHomeDeploymentHealth} from '../../home/vendor/vendor-home';
import {
  filterOpenSupportBundles,
  toCustomerUpdates,
  toVendorHomeDeploymentHealth,
} from '../../home/vendor/vendor-home-mappers';
import {ApplicationEntitlement} from '../../types/application-entitlement';
import {LatestArtifactPull} from '../../types/artifact-version-pull';
import {License} from '../../types/license';
import {LicenseTemplate} from '../../types/license-template';
import {SupportBundle, SupportBundleStatus} from '../../types/support-bundle';
import {CustomerDetailTabId} from './detail/customer-detail-navigation';

const licenseExpiryWindowDays = 7;

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

export interface CustomerListDeployments extends VendorHomeDeploymentHealth {
  /** The deployments that are progressing or have not reported a status. */
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

export interface CustomerListLicenseSources {
  licenses: License[];
  templates: LicenseTemplate[];
}

/**
 * Decided by the features when the customer has only one of deployment targets and artifacts, and by whether it
 * deployed or pulled anything when it has both, since every customer gets both by default.
 */
function toCustomerManagement(
  customer: CustomerOrganizationWithUsage,
  health: VendorHomeDeploymentHealth | undefined,
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

/**
 * Expired when any license key or entitlement has expired, expiring when any expires within the window and valid
 * otherwise. The window of a key created from a template is at least the template's grace period, because the
 * vendor billing sets such a key to expire that long after the billing period ends, so a key inside it has already
 * missed its renewal.
 */
export function toCustomerLicenseStatus(
  license: License | undefined,
  templates: LicenseTemplate[],
  now: Dayjs = dayjs()
): CustomerLicenseStatus {
  if (license === undefined) {
    return 'no_license';
  }
  const gracePeriodDays = new Map(templates.map((template) => [template.id, template.expirationGracePeriodDays]));
  const expirations = [
    ...license.licenseKeys.map((key) => ({
      expiresAt: key.expiresAt,
      windowDays: Math.max(
        licenseExpiryWindowDays,
        (key.licenseTemplateId && gracePeriodDays.get(key.licenseTemplateId)) || 0
      ),
    })),
    ...[...license.applicationEntitlements, ...license.artifactEntitlements].map((entitlement) => ({
      expiresAt: entitlement.expiresAt,
      windowDays: licenseExpiryWindowDays,
    })),
  ];
  if (expirations.length === 0) {
    return 'no_license';
  }
  const expiring = expirations.filter((it) => it.expiresAt !== undefined);
  if (expiring.some(({expiresAt}) => !now.isBefore(expiresAt))) {
    return 'expired';
  } else if (expiring.some(({expiresAt, windowDays}) => !now.add(windowDays, 'day').isBefore(expiresAt))) {
    return 'expiring';
  } else {
    return 'valid';
  }
}

export function toCustomerListRows(
  customers: CustomerOrganizationWithUsage[],
  deployments: CustomerListDeploymentSources | undefined,
  licenses: CustomerListLicenseSources | undefined,
  supportBundles: SupportBundle[] | undefined,
  now: Dayjs = dayjs()
): CustomerListRow[] {
  const licenseByCustomer = new Map(licenses?.licenses.map((license) => [license.customerOrganization.id, license]));
  return customers.map((customer): CustomerListRow => {
    const health =
      deployments &&
      toVendorHomeDeploymentHealth(
        deployments.targets.filter((target) => target.customerOrganization?.id === customer.id)
      );
    return {
      id: customer.id,
      name: customer.name,
      imageUrl: customer.imageUrl,
      management: toCustomerManagement(customer, health, deployments?.pulls),
      updatesTab: (health?.total ?? 0) > 0 ? 'deployments' : 'downloads',
      license: licenses && toCustomerLicenseStatus(licenseByCustomer.get(customer.id), licenses.templates, now),
      deployments: health && {
        ...health,
        noStatus: health.total - health.healthy - health.running - health.error - health.stale,
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
