import {Feature} from './organization';
import {FEATURE_UPGRADE_PLANS, PlanManagedFeature, UPGRADE_PLANS, UpgradePlan} from './subscription';

/** Locks until the organization has any of the features. The plan to upgrade to follows from the features. */
export interface FeatureLock {
  requiresAnyOf: readonly [PlanManagedFeature, ...PlanManagedFeature[]];
}

export function isLocked(lock: FeatureLock | undefined, features: ReadonlySet<Feature>): boolean {
  return lock !== undefined && !lock.requiresAnyOf.some((feature) => features.has(feature));
}

/** The lowest plan that grants any of the features of the lock. */
export function upgradePlan(lock: FeatureLock): UpgradePlan {
  return lock.requiresAnyOf
    .map((feature) => FEATURE_UPGRADE_PLANS[feature])
    .reduce((lowest, plan) => (UPGRADE_PLANS.indexOf(plan) < UPGRADE_PLANS.indexOf(lowest) ? plan : lowest));
}
