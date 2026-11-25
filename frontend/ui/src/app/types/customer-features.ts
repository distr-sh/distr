import {CustomerOrganizationFeature} from '@distr-sh/distr-sdk';
import {isNonEmpty} from '../../util/arrays';
import {FeatureLock} from './feature-lock';
import {PlanManagedFeature} from './subscription';

/**
 * The organization feature a vendor needs to hand out each customer feature, or null for one every vendor can. Mirrors
 * CustomerOrganizationFeatureRequirements in internal/types/customer_organization_feature.go: change both together.
 */
export const CUSTOMER_FEATURE_REQUIREMENTS = {
  deployment_targets: null,
  artifacts: null,
  alerts: 'notifications',
  update_notifications: 'notifications',
  support_bundles: 'support',
  vulnerabilities: 'vulnerabilities',
  oidc_providers: 'custom_oidc_providers',
  settings: 'customer_settings',
} as const satisfies Record<CustomerOrganizationFeature, PlanManagedFeature | null>;

export interface CustomerFeatureDefinition {
  feature: CustomerOrganizationFeature;
  label: string;
  /** The feature this one builds on: enabling this one enables it as well, disabling it disables this one. */
  requires?: CustomerOrganizationFeature;
  /** The features this one needs one of: it cannot be enabled without one, disabling the last one disables it. */
  requiresAnyOf?: CustomerOrganizationFeature[];
}

export const customerFeatures: CustomerFeatureDefinition[] = [
  {feature: 'deployment_targets', label: 'Deployments'},
  {feature: 'alerts', label: 'Alerts', requires: 'deployment_targets'},
  {feature: 'artifacts', label: 'Artifacts'},
  {
    feature: 'update_notifications',
    label: 'Update Notifications',
    requiresAnyOf: ['deployment_targets', 'artifacts'],
  },
  {feature: 'support_bundles', label: 'Support'},
  {feature: 'vulnerabilities', label: 'Security'},
  {feature: 'settings', label: 'Advanced Settings'},
  {feature: 'oidc_providers', label: 'Identity Provider', requires: 'settings'},
];

/**
 * Locks what belongs to the given customer features until the organization has a feature that unlocks one of them.
 * Nothing is locked when one of them needs no organization feature at all.
 */
export function customerFeaturesLock(...features: CustomerOrganizationFeature[]): FeatureLock | undefined {
  const required = features.map((feature) => CUSTOMER_FEATURE_REQUIREMENTS[feature]);
  return required.every((it) => it !== null) && isNonEmpty(required) ? {requiresAnyOf: required} : undefined;
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
