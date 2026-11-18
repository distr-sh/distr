import {defineRoutedTabs, featureLock} from '../../../navigation/tabs';
import {customerFeaturesLock} from '../../../types/customer-features';

export type CustomerDetailTabId =
  | 'overview'
  | 'users'
  | 'deployments'
  | 'registry'
  | 'secrets'
  | 'links'
  | 'licenses'
  | 'notifications'
  | 'support'
  | 'settings';

const licensesLock = featureLock('pro', 'licensing');
const notificationsLock = customerFeaturesLock('alerts', 'update_notifications');
const supportLock = customerFeaturesLock('support_bundles');
const settingsLock = customerFeaturesLock('settings');

export const customerDetailTabs = defineRoutedTabs<CustomerDetailTabId>('Customers', [
  {id: 'overview', label: 'Overview'},
  {id: 'users', label: 'Users'},
  {id: 'deployments', label: 'Deployments'},
  {id: 'registry', label: 'Registry'},
  {id: 'secrets', label: 'Secrets'},
  {id: 'links', label: 'Links'},
  {
    id: 'licenses',
    label: 'Licenses',
    visible: (c) => !c.user.isPartner || !licensesLock.when(c),
    locked: licensesLock,
  },
  {
    id: 'notifications',
    label: 'Notifications',
    visible: (c) => !c.user.isPartner || !notificationsLock.when(c),
    locked: notificationsLock,
  },
  {
    id: 'support',
    label: 'Support',
    visible: (c) => !c.user.isPartner || !supportLock.when(c),
    locked: supportLock,
  },
  {
    id: 'settings',
    label: 'Settings',
    visible: (c) => c.user.isAdminOrSuperAdmin && (!c.user.isPartner || !settingsLock.when(c)),
    locked: settingsLock,
  },
]);
