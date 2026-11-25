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
      {id: 'home', label: 'Home', icon: faHome, route: '/home', shortcut: 'h'},
      {
        id: 'deployments',
        label: 'Deployments',
        icon: faDashboard,
        route: '/deployments',
        shortcut: 'd',
        visible: (c) => c.customer.features.has('deployment_targets'),
      },
      {
        id: 'artifacts',
        label: 'Artifacts',
        icon: faBox,
        route: '/artifacts',
        shortcut: 'a',
        visible: (c) => c.customer.features.has('artifacts'),
      },
      {
        id: 'licenses',
        label: 'Licenses',
        icon: faKey,
        route: '/licenses',
        shortcut: 'l',
        visible: (c) => c.customer.hasLicenseKeys,
      },
      {
        id: 'notifications',
        label: 'Notifications',
        icon: faBell,
        route: '/notifications',
        shortcut: 'n',
        visible: (c) =>
          c.features.has('notifications') &&
          (c.customer.features.has('alerts') || c.customer.features.has('update_notifications')),
      },
      {
        id: 'support',
        label: 'Support',
        icon: faLifeRing,
        route: '/support',
        shortcut: 'b',
        visible: (c) => c.features.has('support') && c.customer.features.has('support_bundles'),
      },
      {
        id: 'security',
        label: 'Security',
        icon: faShieldHalved,
        route: '/security',
        shortcut: 'v',
        visible: (c) => c.features.has('vulnerabilities') && c.customer.features.has('vulnerabilities'),
      },
    ],
  },
  {
    id: 'organization',
    label: 'Organization',
    entries: [
      {
        id: 'team',
        label: 'Team',
        icon: faUsers,
        route: '/team',
        shortcut: 't',
        visible: (c) => c.user.isAdmin,
      },
      {
        id: 'secrets',
        label: 'Secrets',
        icon: faAsterisk,
        route: '/secrets',
        shortcut: 'e',
        visible: (c) => c.customer.features.has('deployment_targets'),
      },
      {
        id: 'settings',
        label: 'Settings',
        icon: faGear,
        route: '/settings/customer',
        shortcut: 's',
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
