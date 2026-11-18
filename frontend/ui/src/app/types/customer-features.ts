import {CustomerOrganizationFeature} from '@distr-sh/distr-sdk';
import {featureLock, proLock, TabLock} from '../navigation/tabs';

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
  {feature: 'alerts', label: 'Alerts', locked: proLock, requires: 'deployment_targets'},
  {feature: 'artifacts', label: 'Artifacts'},
  {
    feature: 'update_notifications',
    label: 'Update Notifications',
    locked: proLock,
    requiresAnyOf: ['deployment_targets', 'artifacts'],
  },
  {feature: 'support_bundles', label: 'Support', locked: proLock},
  {feature: 'vulnerabilities', label: 'Security', locked: featureLock('business', 'vulnerabilities')},
  {feature: 'settings', label: 'Settings', locked: featureLock('business', 'customer_settings')},
  {
    feature: 'oidc_providers',
    label: 'Identity Provider',
    locked: featureLock('business', 'custom_oidc_providers'),
    requires: 'settings',
  },
];

/** Locks what belongs to the given customer features while the organization's plan lets it grant none of them. */
export function customerFeaturesLock(...features: CustomerOrganizationFeature[]): TabLock {
  const locks = customerFeatures.filter((it) => features.includes(it.feature)).map((it) => it.locked);
  return {
    plan: locks.some((it) => it?.plan === 'pro') ? 'pro' : 'business',
    when: (c) => locks.every((it) => it !== undefined && it.when(c)),
  };
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
