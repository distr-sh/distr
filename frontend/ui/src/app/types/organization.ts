import {BaseModel, Named, UserRole} from '@distr-sh/distr-sdk';
import dayjs from 'dayjs';
import {SubscriptionType} from './subscription';

export type Feature =
  | 'licensing'
  | 'pre_post_scripts'
  | 'artifact_version_mutable'
  | 'vendor_billing'
  | 'deployment_logs_after'
  | 'partner_management';

export interface SubscriptionLimits {
  maxCustomerOrganizations: number;
  maxUsersPerCustomerOrganization: number;
  maxDeploymentsPerCustomerOrganization: number;
}

export interface UpdateOrganizationRequest {
  name: string;
  slug?: string;
  preConnectScript?: string;
  postConnectScript?: string;
  connectScriptIsSudo: boolean;
  artifactVersionMutable: boolean;
  prePostScriptsEnabled: boolean;
  pageTitle?: string;
  faviconImageId?: string;
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
  currentBillableUserAccountCount: number;
  currentCustomerOrganizationCount: number;
  preConnectScript?: string;
  postConnectScript?: string;
  connectScriptIsSudo: boolean;
  stripeWebhookSecretConfigured: boolean;
  pageTitle?: string;
  faviconImageId?: string;
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
  return org.subscriptionType !== 'community' && dayjs(org.subscriptionEndsAt).isBefore();
}
