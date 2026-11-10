import {faBell} from '@fortawesome/free-solid-svg-icons';
import {NavigationChildDefinition, NavigationGroupDefinition} from '../navigation';

const alerts: NavigationChildDefinition = {
  id: 'alert-configurations',
  label: 'Alerts',
  route: '/notifications/alert-configurations',
};
const updates: NavigationChildDefinition = {
  id: 'update-notifications',
  label: 'Updates',
  route: '/notifications/updates',
};
const history: NavigationChildDefinition = {
  id: 'notification-history',
  label: 'History',
  route: '/notifications/history',
};

export const vendorNotificationsGroup: NavigationGroupDefinition = {
  id: 'notifications',
  label: 'Notifications',
  icon: faBell,
  locked: {plan: 'pro', when: (c) => !c.features.has('notifications')},
  children: [alerts, updates, history],
};

export const customerNotificationsGroup: NavigationGroupDefinition = {
  id: 'notifications',
  label: 'Notifications',
  icon: faBell,
  visible: (c) =>
    c.features.has('notifications') &&
    (c.customer.features.has('alerts') || c.customer.features.has('update_notifications')),
  children: [
    {...alerts, visible: (c) => c.customer.features.has('alerts')},
    {...updates, visible: (c) => c.customer.features.has('update_notifications')},
    history,
  ],
};
