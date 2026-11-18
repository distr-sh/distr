import {CustomerOrganizationFeature} from '@distr-sh/distr-sdk';
import {proSubscriptionLock, TabLock} from '../navigation/tabs';
import {UPGRADE_PLANS} from './subscription';

export interface CustomerFeatureDefinition {
  feature: CustomerOrganizationFeature;
  label: string;
  locked?: TabLock;
  /** The feature this one builds on: enabling this one enables it as well, disabling it disables this one. */
  requires?: CustomerOrganizationFeature;
  /** The features this one needs one of: it cannot be enabled without one, disabling the last one disables it. */
  requiresAnyOf?: CustomerOrganizationFeature[];
}

export const customerFeatures: CustomerFeatureDefinition[] = [
  {feature: 'deployment_targets', label: 'Deployments'},
  {feature: 'alerts', label: 'Alerts', locked: proSubscriptionLock, requires: 'deployment_targets'},
  {feature: 'artifacts', label: 'Artifacts'},
  {
    feature: 'update_notifications',
    label: 'Update Notifications',
    locked: proSubscriptionLock,
    requiresAnyOf: ['deployment_targets', 'artifacts'],
  },
  {feature: 'support_bundles', label: 'Support', locked: proSubscriptionLock},
  {
    feature: 'vulnerabilities',
    label: 'Security',
    locked: {requiresAnyOf: ['vulnerabilities'], upgradePlan: 'business'},
  },
  {feature: 'settings', label: 'Settings', locked: {requiresAnyOf: ['customer_settings'], upgradePlan: 'business'}},
  {
    feature: 'oidc_providers',
    label: 'Identity Provider',
    locked: {requiresAnyOf: ['custom_oidc_providers'], upgradePlan: 'business'},
    requires: 'settings',
  },
];

/**
 * Locks what belongs to the given customer features until the organization has a feature that unlocks one of them,
 * naming the lowest plan among them for the upgrade.
 */
export function customerFeaturesLock(...features: CustomerOrganizationFeature[]): TabLock | undefined {
  const locks = customerFeatures.filter((it) => features.includes(it.feature)).map((it) => it.locked);
  if (!locks.every((it) => it !== undefined)) {
    return undefined;
  } else if (locks.includes(proSubscriptionLock)) {
    return proSubscriptionLock;
  }
  const upgradePlan = UPGRADE_PLANS.find((plan) => locks.some((it) => it.upgradePlan === plan));
  return upgradePlan && {requiresAnyOf: locks.flatMap((it) => it.requiresAnyOf), upgradePlan};
}

export function switchCustomerFeature(
  features: CustomerOrganizationFeature[],
  feature: CustomerOrganizationFeature,
  enable: boolean
): CustomerOrganizationFeature[] {
  const result = new Set(features);
  if (enable) {
    result.add(feature);
    const requires = customerFeatures.find((it) => it.feature === feature)?.requires;
    if (requires) {
      result.add(requires);
    }
  } else {
    result.delete(feature);
    for (const dependent of customerFeatures) {
      if (
        dependent.requires === feature ||
        (dependent.requiresAnyOf?.includes(feature) && !dependent.requiresAnyOf.some((it) => result.has(it)))
      ) {
        result.delete(dependent.feature);
      }
    }
  }
  return Array.from(result);
}

/**
 * Explains why the feature cannot be enabled yet, or returns undefined when it can. A missing `requires` blocks it
 * only for a caller that does not enable the required feature along with it.
 */
export function unmetCustomerFeatureRequirement(
  features: CustomerOrganizationFeature[],
  feature: CustomerOrganizationFeature,
  {enablesRequired}: {enablesRequired: boolean}
): string | undefined {
  const definition = customerFeatures.find((it) => it.feature === feature);
  const requirements = [
    ...(definition?.requiresAnyOf ? [definition.requiresAnyOf] : []),
    ...(definition?.requires && !enablesRequired ? [[definition.requires]] : []),
  ];
  const unmet = requirements.find((anyOf) => !anyOf.some((it) => features.includes(it)));
  return unmet ? customerFeatureRequiredMessage(unmet) : undefined;
}

export function customerFeatureRequiredMessage(anyOf: CustomerOrganizationFeature[]): string {
  const labels = customerFeatures.filter((it) => anyOf.includes(it.feature)).map((it) => it.label);
  return `Requires the ${labels.join(' or ')} feature to be enabled`;
}
