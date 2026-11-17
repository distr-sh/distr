export interface DeploymentHealth {
  total: number;
  healthy: number;
  running: number;
  error: number;
  stale: number;
  progressing: number;
}

export interface VendorHomeUpdates {
  totalCustomers: number;
  outdatedCustomers: number;
  outdatedDeployments: number;
}

export interface VendorHomeLicenses {
  expiredCustomerIds: string[];
  expiringCustomerIds: string[];
  unlicensedCustomerIds: string[];
}

export interface VendorHomeAdvisoryCounts {
  total: number;
  open: number;
  triage: number;
  draft: number;
  published: number;
}

export interface VendorHomeCustomer {
  id: string;
  name: string;
  imageUrl?: string;
  usage: {type: 'deployments' | 'downloads' | 'users'; count: number};
  openItems: number;
}
