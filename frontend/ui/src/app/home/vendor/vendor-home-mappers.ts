import {CustomerOrganization, DeploymentTarget} from '@distr-sh/distr-sdk';
import {
  DeploymentDisplayStatus,
  deploymentDisplayStatus,
} from '../../deployments/deployment-target-card/deployment-status-text.component';
import {VendorHomeCustomer, VendorHomeDeploymentHealth} from './vendor-home';

/** The status filter the deployments card links to, which has to show exactly the error and stale deployments. */
export const vendorHomeAttentionStatuses: readonly DeploymentDisplayStatus[] = ['error', 'stale'];

export function toVendorHomeDeploymentHealth(targets: DeploymentTarget[]): VendorHomeDeploymentHealth {
  const statuses = targets.flatMap((target) => target.deployments.map(deploymentDisplayStatus));
  const count = (status: DeploymentDisplayStatus) => statuses.filter((it) => it === status).length;
  return {healthy: count('healthy'), running: count('running'), error: count('error'), stale: count('stale')};
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
