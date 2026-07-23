export const UNLIMITED_QTY = -1;
export type SubscriptionType = 'community' | 'pro' | 'business' | 'enterprise' | 'trial';

export type SubscriptionPeriod = 'monthly' | 'yearly';

export interface SubscriptionLimits {
  maxCustomerOrganizations: number;
  maxUsersPerCustomerOrganization: number;
  maxDeploymentsPerCustomerOrganization: number;
  logQueryWindowSeconds: number;
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
  hasApplicationEntitlements: boolean;
  hasArtifactEntitlements: boolean;
  hasNonAdminRoles: boolean;
  hasAlertConfigurations: boolean;
  limits: {[key in SubscriptionType]: SubscriptionLimits};
}

export interface CheckoutRequest {
  subscriptionType: SubscriptionType;
  subscriptionPeriod: SubscriptionPeriod;
  subscriptionUserAccountQuantity: number;
  subscriptionCustomerOrganizationQuantity: number;
}

export interface UpdateSubscriptionRequest {
  // Optionally switches the subscription to a different plan (currently only pro → business)
  subscriptionType?: SubscriptionType;
  subscriptionUserAccountQuantity: number;
  subscriptionCustomerOrganizationQuantity: number;
}
