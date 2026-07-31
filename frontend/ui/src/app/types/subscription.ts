export const UNLIMITED_QTY = -1;
export type SubscriptionType = 'community' | 'pro' | 'business' | 'enterprise' | 'trial';

// Subscription types without access to paid features. Feature gating is expressed by
// excluding these types rather than by listing the paid ones, so a newly introduced plan
// has access by default instead of silently losing it.
export const NON_PRO_SUBSCRIPTION_TYPES: SubscriptionType[] = ['community'];

// Subscription types that are not a paying subscription, i.e. where upgrade prompts apply.
export const NON_PAYING_SUBSCRIPTION_TYPES: SubscriptionType[] = [...NON_PRO_SUBSCRIPTION_TYPES, 'trial'];

export function isProSubscription(subscriptionType: SubscriptionType | undefined): boolean {
  return subscriptionType !== undefined && !NON_PRO_SUBSCRIPTION_TYPES.includes(subscriptionType);
}

export function isPayingSubscription(subscriptionType: SubscriptionType | undefined): boolean {
  return subscriptionType !== undefined && !NON_PAYING_SUBSCRIPTION_TYPES.includes(subscriptionType);
}

export type SubscriptionPeriod = 'monthly' | 'yearly';

export interface SubscriptionLimits {
  maxCustomerOrganizations: number;
  maxUsersPerCustomerOrganization: number;
  maxDeploymentsPerCustomerOrganization: number;
  // Reported for display purposes only, this limit is not enforced.
  maxRegistryStorageBytes: number;
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
  currentRegistryStorageBytes: number;
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
