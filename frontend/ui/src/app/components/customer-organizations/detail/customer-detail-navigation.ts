import {defineRoutedTabs} from '../../../navigation/tabs';

export type CustomerDetailTabId =
  | 'overview'
  | 'deployments'
  | 'downloads'
  | 'licenses'
  | 'users'
  | 'secrets'
  | 'support'
  | 'notifications'
  | 'settings';

export const customerDetailTabs = defineRoutedTabs<CustomerDetailTabId>('Customers', [
  {id: 'overview', label: 'Overview'},
  {id: 'deployments', label: 'Deployments'},
  {id: 'downloads', label: 'Downloads'},
  {
    id: 'licenses',
    label: 'Licenses',
    visible: (c) => !c.user.isPartner || c.features.has('licensing'),
    locked: {plan: 'pro', when: (c) => !c.features.has('licensing')},
  },
  {id: 'users', label: 'Users'},
  {id: 'secrets', label: 'Secrets'},
  {
    id: 'support',
    label: 'Support',
    visible: (c) => !c.user.isPartner || c.organization.isPro,
    locked: {plan: 'pro', when: (c) => !c.organization.isPro},
  },
  {
    id: 'notifications',
    label: 'Notifications',
    visible: (c) => !c.user.isPartner,
    locked: {plan: 'pro', when: (c) => !c.organization.isPro},
  },
  {id: 'settings', label: 'Settings'},
]);
