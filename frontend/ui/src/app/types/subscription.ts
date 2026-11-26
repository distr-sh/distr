import dayjs from 'dayjs';

export const UNLIMITED_QTY = -1;
export type SubscriptionType = 'community' | 'pro' | 'business' | 'enterprise' | 'trial';

/** The plans a feature can ask an organization to upgrade to, from the lowest to the highest. */
export const UPGRADE_PLANS = ['pro', 'business'] as const satisfies readonly SubscriptionType[];
export type UpgradePlan = (typeof UPGRADE_PLANS)[number];

/**
 * The lowest plan that grants each plan-managed feature. Mirrors ProFeatures and BusinessFeatures in
 * internal/types/types.go: change both together, so that a lock never offers a plan that does not unlock it.
 */
export const FEATURE_UPGRADE_PLANS = {
  licensing: 'pro',
  support: 'pro',
  notifications: 'pro',
  user_roles: 'pro',
  partner_management: 'business',
  custom_domains: 'business',
  vulnerabilities: 'business',
  custom_emails: 'business',
  custom_oidc_providers: 'business',
  auto_updates: 'business',
  customer_settings: 'business',
} as const satisfies Record<string, UpgradePlan>;

export type PlanManagedFeature = keyof typeof FEATURE_UPGRADE_PLANS;

// Subscription types that are not a paying subscription, i.e. where upgrade prompts apply.
export const NON_PAYING_SUBSCRIPTION_TYPES: SubscriptionType[] = ['community', 'trial'];

export function isPayingSubscription(subscriptionType: SubscriptionType | undefined): boolean {
  return subscriptionType !== undefined && !NON_PAYING_SUBSCRIPTION_TYPES.includes(subscriptionType);
}

export function isExpiredSubscription(subscriptionType: SubscriptionType, subscriptionEndsAt?: string): boolean {
  return subscriptionType !== 'community' && !!subscriptionEndsAt && dayjs(subscriptionEndsAt).isBefore();
}

// the plans from the most to the least capable, for lists that show organizations of several plans
const SUBSCRIPTION_ORDER: SubscriptionType[] = ['enterprise', 'business', 'pro', 'trial', 'community'];

export function subscriptionRank(subscriptionType: SubscriptionType, subscriptionEndsAt?: string): number {
  if (isExpiredSubscription(subscriptionType, subscriptionEndsAt)) {
    // an expired subscription grants nothing, whatever plan it was on
    return SUBSCRIPTION_ORDER.length + 1;
  }
  const rank = SUBSCRIPTION_ORDER.indexOf(subscriptionType);
  // a plan introduced after this list ranks below the ones it knows, but above anything expired
  return rank === -1 ? SUBSCRIPTION_ORDER.length : rank;
}

export type SubscriptionPeriod = 'monthly' | 'yearly';

export function subscriptionPrice(
  subscriptionType: SubscriptionType,
  subscriptionPeriod: SubscriptionPeriod,
  userQty: number,
  customerQty: number
): number {
  let userPrice = 0;
  let customerPrice = 0;

  if (subscriptionType === 'pro') {
    userPrice = subscriptionPeriod === 'monthly' ? 29 : 288;
    customerPrice = subscriptionPeriod === 'monthly' ? 69 : 672;
  } else if (subscriptionType === 'business') {
    userPrice = subscriptionPeriod === 'monthly' ? 39 : 384;
    customerPrice = subscriptionPeriod === 'monthly' ? 159 : 1536;
  }
  return userPrice * userQty + customerPrice * customerQty;
}

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
  hasStripeSubscription: boolean;
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
