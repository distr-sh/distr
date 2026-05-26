import {AccountRole, BaseModel, Named} from '@distr-sh/distr-sdk';
import {SubscriptionType} from './subscription';

export type Feature =
  | 'licensing'
  | 'pre_post_scripts'
  | 'artifact_version_mutable'
  | 'vendor_billing'
  | 'deployment_logs_after';

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

export interface OrganizationWithRole extends Organization {
  accountRole: AccountRole;
  /**
   * @deprecated Use {@link accountRole}. This field is a server-side alias kept for
   * backwards compatibility and may be removed in a future release.
   */
  userRole: AccountRole;
  customerOrganizationId?: string;
  customerOrganizationName?: string;
  joinedOrgAt: string;
}
