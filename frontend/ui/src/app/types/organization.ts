import {BaseModel, Named, UserRole} from '@distr-sh/distr-sdk';
import {isExpiredSubscription, SubscriptionType} from './subscription';

export type Feature =
  | 'licensing'
  | 'pre_post_scripts'
  | 'artifact_version_mutable'
  | 'vendor_billing'
  | 'deployment_logs_after'
  | 'partner_management'
  | 'custom_domains'
  | 'vulnerabilities'
  | 'custom_emails'
  | 'custom_oidc_providers'
  | 'auto_updates';

export interface SubscriptionLimits {
  maxCustomerOrganizations: number;
  maxUsersPerCustomerOrganization: number;
  maxDeploymentsPerCustomerOrganization: number;
  // Reported for display purposes only, this limit is not enforced.
  maxRegistryStorageBytes: number;
  logQueryWindowSeconds: number;
}

export interface UpdateOrganizationRequest {
  name: string;
  slug?: string;
  dockerPreConnectScript?: string;
  dockerPostConnectScript?: string;
  dockerConnectScriptIsSudo: boolean;
  kubernetesPreConnectScript?: string;
  kubernetesPostConnectScript?: string;
  artifactVersionMutable: boolean;
  prePostScriptsEnabled: boolean;
  controllerImageRepository?: string;
}

export interface Organization extends BaseModel, Named {
  name: string;
  slug?: string;
  features: Feature[];
  subscriptionType: SubscriptionType;
  subscriptionLimits: SubscriptionLimits;
  subscriptionEndsAt?: string;
  subscriptionCustomerOrganizationQuantity: number;
  subscriptionUserAccountQuantity: number;
  currentBillableUserAccountCount: number;
  currentCustomerOrganizationCount: number;
  dockerPreConnectScript?: string;
  dockerPostConnectScript?: string;
  dockerConnectScriptIsSudo: boolean;
  kubernetesPreConnectScript?: string;
  kubernetesPostConnectScript?: string;
  stripeWebhookSecretConfigured: boolean;
  controllerImageRepository?: string;
  defaultControllerImageRepository: string;
}

export interface OrganizationWithUserRole extends Organization {
  userRole: UserRole;
  customerOrganizationId?: string;
  customerOrganizationName?: string;
  partnerOrganizationId?: string;
  partnerOrganizationName?: string;
  joinedOrgAt: string;
}

export function isSubscriptionExpired(org: Organization): boolean {
  return isExpiredSubscription(org.subscriptionType, org.subscriptionEndsAt);
}
