import {defineRoutedTabs} from '../../../navigation/tabs';

export type CustomerDetailTabId =
  | 'overview'
  | 'deployments'
  | 'artifacts'
  | 'licenses'
  | 'users'
  | 'secrets'
  | 'support'
  | 'notifications'
  | 'links'
  | 'settings';

export const customerDetailTabs = defineRoutedTabs<CustomerDetailTabId>('Customers', [
  {id: 'overview', label: 'Overview'},
  {id: 'deployments', label: 'Deployments'},
  {id: 'artifacts', label: 'Artifacts'},
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
    visible: (c) => !c.user.isPartner || c.organization.isPro,
    locked: {plan: 'pro', when: (c) => !c.organization.isPro},
  },
  {id: 'links', label: 'Links'},
  {
    id: 'settings',
    label: 'Settings',
    visible: (c) => c.user.isAdminOrSuperAdmin && (!c.user.isPartner || c.features.has('customer_settings')),
    locked: {plan: 'business', when: (c) => !c.features.has('customer_settings')},
  },
]);
