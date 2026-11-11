import {
  faArrowUpRightFromSquare,
  faAsterisk,
  faBell,
  faBox,
  faDashboard,
  faGear,
  faHome,
  faKey,
  faLifeRing,
  faShieldHalved,
  faUsers,
} from '@fortawesome/free-solid-svg-icons';
import {NavigationDefinition} from './navigation';

export const customerNavigation: NavigationDefinition = [
  {
    id: 'software',
    label: 'Software',
    entries: [
      {id: 'home', label: 'Home', icon: faHome, route: '/home'},
      {
        id: 'deployments',
        label: 'Deployments',
        icon: faDashboard,
        route: '/deployments',
        visible: (c) => c.customer.features.has('deployment_targets'),
      },
      {
        id: 'artifacts',
        label: 'Artifacts',
        icon: faBox,
        route: '/artifacts',
        visible: (c) => c.customer.features.has('artifacts'),
      },
      {
        id: 'licenses',
        label: 'Licenses',
        icon: faKey,
        route: '/license-keys',
        visible: (c) => c.customer.hasLicenseKeys,
      },
      {
        id: 'notifications',
        label: 'Notifications',
        icon: faBell,
        visible: (c) =>
          c.features.has('notifications') &&
          (c.customer.features.has('alerts') || c.customer.features.has('update_notifications')),
        children: [
          {
            id: 'alert-configurations',
            label: 'Alerts',
            route: '/notifications/alert-configurations',
            visible: (c) => c.customer.features.has('alerts'),
          },
          {
            id: 'update-notifications',
            label: 'Updates',
            route: '/notifications/updates',
            visible: (c) => c.customer.features.has('update_notifications'),
          },
          {id: 'notification-history', label: 'History', route: '/notifications/history'},
        ],
      },
      {
        id: 'support',
        label: 'Support',
        icon: faLifeRing,
        route: '/support',
        visible: (c) => c.features.has('support_bundles') && c.customer.features.has('support_bundles'),
      },
      {
        id: 'security',
        label: 'Security',
        icon: faShieldHalved,
        route: '/security',
        visible: (c) => c.features.has('vulnerabilities') && c.customer.features.has('vulnerabilities'),
      },
    ],
  },
  {
    id: 'organization',
    label: 'Organization',
    entries: [
      {id: 'users', label: 'Users', icon: faUsers, route: '/users', visible: (c) => c.user.isAdmin},
      {
        id: 'secrets',
        label: 'Secrets',
        icon: faAsterisk,
        route: '/secrets',
        visible: (c) => c.customer.features.has('deployment_targets'),
      },
      {
        id: 'settings',
        label: 'Settings',
        icon: faGear,
        route: '/settings/customer',
        visible: (c) => c.user.isAdmin && c.customer.features.has('settings'),
      },
    ],
  },
  {
    id: 'links',
    label: 'Links',
    entries: (c) =>
      c.customer.sidebarLinks.map((link) => ({
        id: link.id,
        label: link.name,
        icon: faArrowUpRightFromSquare,
        href: link.link,
      })),
  },
];
