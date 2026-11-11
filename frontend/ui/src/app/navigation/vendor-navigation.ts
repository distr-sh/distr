import {
  faAddressBook,
  faAsterisk,
  faBell,
  faBox,
  faBoxesStacked,
  faBuilding,
  faCube,
  faGear,
  faHome,
  faKey,
  faLifeRing,
  faLightbulb,
  faPalette,
  faShieldHalved,
  faUsers,
} from '@fortawesome/free-solid-svg-icons';
import {NavigationDefinition} from './navigation';

export const vendorNavigation: NavigationDefinition = [
  {
    id: 'distribution',
    label: 'Distribution',
    entries: [
      {
        id: 'tutorials',
        label: 'Tutorials',
        icon: faLightbulb,
        route: '/tutorials',
        visible: (c) => c.user.isAdmin && !c.organization.hasSubscription,
        indicator: (c) => c.organization.hasUnstartedTutorials,
      },
      {id: 'home', label: 'Home', icon: faHome, route: '/home'},
      {id: 'applications', label: 'Applications', icon: faCube, route: '/applications'},
      {id: 'deployments', label: 'Deployments', icon: faBoxesStacked, route: '/deployments'},
      {
        id: 'registry',
        label: 'Registry',
        icon: faBox,
        children: [
          {id: 'artifacts', label: 'Artifacts', route: '/artifacts'},
          {id: 'artifact-pulls', label: 'Downloads', route: '/artifact-pulls'},
        ],
      },
      {
        id: 'licenses',
        label: 'Licenses',
        icon: faKey,
        route: '/licenses',
        locked: {plan: 'pro', when: (c) => !c.features.has('licensing')},
      },
      {
        id: 'notifications',
        label: 'Notifications',
        icon: faBell,
        locked: {plan: 'pro', when: (c) => !c.features.has('notifications')},
        children: [
          {id: 'alert-configurations', label: 'Alerts', route: '/notifications/alert-configurations'},
          {id: 'update-notifications', label: 'Updates', route: '/notifications/updates'},
          {id: 'notification-history', label: 'History', route: '/notifications/history'},
        ],
      },
    ],
  },
  {
    id: 'management',
    label: 'Management',
    entries: [
      {id: 'customers', label: 'Customers', icon: faAddressBook, route: '/customers'},
      {
        id: 'partners',
        label: 'Partners',
        icon: faBuilding,
        route: '/partners',
        visible: (c) => c.features.has('partner_management'),
      },
      {
        id: 'support',
        label: 'Support',
        icon: faLifeRing,
        route: '/support',
        locked: {plan: 'pro', when: (c) => !c.features.has('support_bundles')},
      },
      {
        id: 'advisories',
        label: 'Security Advisories',
        icon: faShieldHalved,
        route: '/advisories',
        locked: {plan: 'business', when: (c) => !c.features.has('vulnerabilities')},
      },
      {id: 'users', label: 'Users', icon: faUsers, route: '/users', visible: (c) => c.user.isAdminOrSuperAdmin},
      {id: 'secrets', label: 'Secrets', icon: faAsterisk, route: '/secrets'},
      {
        id: 'customer-portal',
        label: 'Customer Portal',
        icon: faPalette,
        route: '/customer-portal',
        visible: (c) => c.user.isSuperAdmin || c.user.isReadWriteOrAdmin,
      },
      {
        id: 'settings',
        label: 'Settings',
        icon: faGear,
        route: '/settings/organization',
        visible: (c) => c.user.isAdminOrSuperAdmin,
      },
    ],
  },
];
