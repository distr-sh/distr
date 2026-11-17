import {CustomerOrganizationFeature} from '@distr-sh/distr-sdk';
import {Feature} from './organization';

export interface CustomerFeatureDefinition {
  feature: CustomerOrganizationFeature;
  label: string;
  /** The feature this one builds on: enabling this one enables it as well, disabling it disables this one. */
  requires?: CustomerOrganizationFeature;
  /** The features this one needs one of: it cannot be enabled without one, disabling the last one disables it. */
  requiresAnyOf?: CustomerOrganizationFeature[];
  requiresPro?: boolean;
  /** The organization feature that unlocks a Business customer feature, which decides instead of the plan. */
  unlockedBy?: Feature;
}

export const customerFeatures: CustomerFeatureDefinition[] = [
  {feature: 'deployment_targets', label: 'Deployments'},
  {feature: 'alerts', label: 'Alerts', requires: 'deployment_targets', requiresPro: true},
  {feature: 'artifacts', label: 'Artifacts'},
  {
    feature: 'update_notifications',
    label: 'Update Notifications',
    requiresAnyOf: ['deployment_targets', 'artifacts'],
    requiresPro: true,
  },
  {feature: 'support_bundles', label: 'Support', requiresPro: true},
  {feature: 'vulnerabilities', label: 'Security', unlockedBy: 'vulnerabilities'},
  {feature: 'settings', label: 'Settings', unlockedBy: 'customer_settings'},
  {
    feature: 'oidc_providers',
    label: 'Identity Provider',
    requires: 'settings',
    unlockedBy: 'custom_oidc_providers',
  },
];

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
