import {defineRoutedTabs} from '../../../navigation/tabs';

export type CustomerDetailTabId =
  | 'overview'
  | 'users'
  | 'deployments'
  | 'artifacts'
  | 'secrets'
  | 'links'
  | 'licenses'
  | 'notifications'
  | 'support'
  | 'settings';

export const customerDetailTabs = defineRoutedTabs<CustomerDetailTabId>('Customers', [
  {id: 'overview', label: 'Overview'},
  {id: 'users', label: 'Users'},
  {id: 'deployments', label: 'Deployments'},
  {id: 'artifacts', label: 'Artifacts'},
  {id: 'secrets', label: 'Secrets'},
  {id: 'links', label: 'Links'},
  {
    id: 'licenses',
    label: 'Licenses',
    visible: (c) => !c.user.isPartner || c.features.has('licensing'),
    locked: {plan: 'pro', when: (c) => !c.features.has('licensing')},
  },
  {
    id: 'notifications',
    label: 'Notifications',
    visible: (c) => !c.user.isPartner || c.organization.isPro,
    locked: {plan: 'pro', when: (c) => !c.organization.isPro},
  },
  {
    id: 'support',
    label: 'Support',
    visible: (c) => !c.user.isPartner || c.organization.isPro,
    locked: {plan: 'pro', when: (c) => !c.organization.isPro},
  },
  {
    id: 'settings',
    label: 'Settings',
    visible: (c) => c.user.isAdminOrSuperAdmin && (!c.user.isPartner || c.features.has('customer_settings')),
    locked: {plan: 'business', when: (c) => !c.features.has('customer_settings')},
  },
]);
