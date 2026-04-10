import {BaseModel, Named, UserRole} from '@distr-sh/distr-sdk';
import {SubscriptionType} from './subscription';

export type Feature =
  | 'licensing'
  | 'pre_post_scripts'
  | 'artifact_version_mutable'
  | 'vendor_billing'
  | 'deployment_logs_after'
  | 'opentofu';

export interface SubscriptionLimits {
  maxCustomerOrganizations: number;
  maxUsersPerCustomerOrganization: number;
  maxDeploymentsPerCustomerOrganization: number;
}

export interface CreateUpdateOrganizationRequest {
  name: string;
  slug?: string;
  preConnectScript?: string;
  postConnectScript?: string;
  connectScriptIsSudo: boolean;
  artifactVersionMutable: boolean;
  prePostScriptsEnabled: boolean;
  openTofuEnabled?: boolean;
}

export interface Organization extends BaseModel, Named {
  name: string;
  slug?: string;
  features: Feature[];
  appDomain?: string;
  registryDomain?: string;
  emailFromAddress?: string;
  subscriptionType: SubscriptionType;
  subscriptionLimits: SubscriptionLimits;
  subscriptionEndsAt?: string;
  subscriptionCustomerOrganizationQuantity: number;
  subscriptionUserAccountQuantity: number;
  preConnectScript?: string;
  postConnectScript?: string;
  connectScriptIsSudo: boolean;
  stripeWebhookSecretConfigured: boolean;
}

export interface OrganizationWithUserRole extends Organization {
  userRole: UserRole;
  customerOrganizationId?: string;
  customerOrganizationName?: string;
  joinedOrgAt: string;
}
