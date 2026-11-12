import {TabItem} from '../../tab-bar.component';

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

export const customerDetailTabs: readonly TabItem<CustomerDetailTabId>[] = [
  {id: 'overview', label: 'Overview'},
  {id: 'deployments', label: 'Deployments'},
  {id: 'downloads', label: 'Downloads'},
  {id: 'licenses', label: 'Licenses'},
  {id: 'users', label: 'Users'},
  {id: 'secrets', label: 'Secrets'},
  {id: 'support', label: 'Support'},
  {id: 'notifications', label: 'Notifications'},
  {id: 'settings', label: 'Settings'},
];
