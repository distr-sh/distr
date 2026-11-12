import {Application, CustomerOrganization, DeploymentTarget} from '@distr-sh/distr-sdk';
import {isDeploymentOutdated} from '../../../util/versions';
import {
  DeploymentDisplayStatus,
  deploymentDisplayStatus,
} from '../../deployments/deployment-target-card/deployment-status-text.component';
import {ApplicationEntitlement} from '../../types/application-entitlement';
import {LatestArtifactPull} from '../../types/artifact-version-pull';
import {VendorHomeCustomer, VendorHomeDeploymentHealth, VendorHomeUpdates} from './vendor-home';

/** The status filter the deployments card links to, which has to show exactly the error and stale deployments. */
export const vendorHomeAttentionStatuses: readonly DeploymentDisplayStatus[] = ['error', 'stale'];

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
