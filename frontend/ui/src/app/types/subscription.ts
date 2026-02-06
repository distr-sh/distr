export const UNLIMITED_QTY = -1;
export type SubscriptionType = 'community' | 'starter' | 'pro' | 'enterprise' | 'trial';

export type SubscriptionPeriod = 'monthly' | 'yearly';

export interface SubscriptionLimits {
  maxCustomerOrganizations: number;
  maxUsersPerCustomerOrganization: number;
  maxDeploymentsPerCustomerOrganization: number;
}

export interface SubscriptionInfo {
  subscriptionType: SubscriptionType;
  subscriptionPeriod: SubscriptionPeriod;
  subscriptionEndsAt: string;
  subscriptionCustomerOrganizationQuantity: number;
  subscriptionUserAccountQuantity: number;
  currentUserAccountCount: number;
  currentCustomerOrganizationCount: number;
  currentMaxUsersPerCustomer: number;
  currentMaxDeploymentTargetsPerCustomer: number;
  hasApplicationLicenses: boolean;
  hasArtifactLicenses: boolean;
  hasNonAdminRoles: boolean;
  hasDeploymentStatusNotificationConfigurations: boolean;
  limits: {[key in SubscriptionType]: SubscriptionLimits};
}

export interface CheckoutRequest {
  subscriptionType: SubscriptionType;
  subscriptionPeriod: SubscriptionPeriod;
  subscriptionUserAccountQuantity: number;
  subscriptionCustomerOrganizationQuantity: number;
}
