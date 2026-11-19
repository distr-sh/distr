import {defineRoutedTabs} from '../../../navigation/tabs';
import {customerFeaturesLock} from '../../../types/customer-features';
import {FeatureLock, isLocked} from '../../../types/feature-lock';

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

const licensesLock: FeatureLock = {requiresAnyOf: ['licensing']};
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
    visible: (c) => !c.user.isPartner || !isLocked(licensesLock, c.features),
    locked: licensesLock,
  },
  {
    id: 'notifications',
    label: 'Notifications',
    visible: (c) => !c.user.isPartner || !isLocked(notificationsLock, c.features),
    locked: notificationsLock,
  },
  {
    id: 'support',
    label: 'Support',
    visible: (c) => !c.user.isPartner || !isLocked(supportLock, c.features),
    locked: supportLock,
  },
  {
    id: 'settings',
    label: 'Settings',
    visible: (c) => c.user.isAdminOrSuperAdmin && (!c.user.isPartner || !isLocked(settingsLock, c.features)),
    locked: settingsLock,
  },
]);
