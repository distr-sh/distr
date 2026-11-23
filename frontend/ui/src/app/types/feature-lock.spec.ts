import {customerFeaturesLock} from './customer-features';
import {upgradePlan} from './feature-lock';

describe('upgradePlan', () => {
  it('should offer the lowest plan that grants any of the features', () => {
    expect(upgradePlan({requiresAnyOf: ['custom_domains', 'licensing']})).toBe('pro');
    expect(upgradePlan({requiresAnyOf: ['custom_domains', 'custom_emails']})).toBe('business');
  });
});

describe('customerFeaturesLock', () => {
  it('should lock until the organization has a feature that unlocks one of the customer features', () => {
    expect(customerFeaturesLock('vulnerabilities', 'support_bundles')).toEqual({
      requiresAnyOf: ['vulnerabilities', 'support'],
    });
  });

  it('should not lock when one of the customer features needs no organization feature', () => {
    expect(customerFeaturesLock('support_bundles', 'deployment_targets')).toBeUndefined();
  });
});
