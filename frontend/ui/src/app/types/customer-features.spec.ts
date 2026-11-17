import {switchCustomerFeature} from './customer-features';

describe('switchCustomerFeature', () => {
  it('should disable update notifications only with the last of deployments and artifacts', () => {
    const features = switchCustomerFeature(
      ['deployment_targets', 'artifacts', 'alerts', 'update_notifications'],
      'deployment_targets',
      false
    );
    expect(features).toEqual(['artifacts', 'update_notifications']);
    expect(switchCustomerFeature(features, 'artifacts', false)).toEqual([]);
  });

  it('should leave update notifications alone when an unrelated feature is disabled', () => {
    expect(switchCustomerFeature(['update_notifications', 'support_bundles'], 'support_bundles', false)).toEqual([
      'update_notifications',
    ]);
  });
});
