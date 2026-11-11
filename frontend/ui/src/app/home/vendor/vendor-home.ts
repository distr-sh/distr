export interface VendorHomeDeploymentHealth {
  healthy: number;
  running: number;
  error: number;
  stale: number;
}

export interface VendorHomeCustomer {
  id: string;
  name: string;
  imageUrl?: string;
  deploymentHealth: VendorHomeDeploymentHealth;
}
