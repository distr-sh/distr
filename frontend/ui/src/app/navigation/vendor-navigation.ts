import {
  faAddressBook,
  faBell,
  faBox,
  faBoxesStacked,
  faBuilding,
  faCube,
  faGear,
  faHome,
  faLifeRing,
  faPalette,
  faShieldHalved,
} from '@fortawesome/free-solid-svg-icons';
import {NavigationDefinition} from './navigation';

export const vendorNavigation: NavigationDefinition = [
  {
    id: 'distribution',
    label: 'Distribution',
    entries: [
      {id: 'home', label: 'Home', icon: faHome, route: '/home', shortcut: 'h'},
      {id: 'applications', label: 'Applications', icon: faCube, route: '/applications', shortcut: 'a'},
      {
        id: 'partners',
        label: 'Partners',
        icon: faBuilding,
        route: '/partners',
        shortcut: 'p',
        visible: (c) => c.features.has('partner_management'),
      },
      {id: 'customers', label: 'Customers', icon: faAddressBook, route: '/customers', shortcut: 'c'},
      {id: 'deployments', label: 'Deployments', icon: faBoxesStacked, route: '/deployments', shortcut: 'd'},
      {id: 'registry', label: 'Registry', icon: faBox, route: '/registry', shortcut: 'r'},
      {
        id: 'notifications',
        label: 'Notifications',
        icon: faBell,
        route: '/notifications',
        shortcut: 'n',
        locked: {requiresAnyOf: ['notifications']},
      },
      {
        id: 'support',
        label: 'Support',
        icon: faLifeRing,
        route: '/support',
        shortcut: 'b',
        locked: {requiresAnyOf: ['support']},
      },
      {
        id: 'security',
        label: 'Security',
        icon: faShieldHalved,
        route: '/security',
        shortcut: 'v',
        locked: {requiresAnyOf: ['vulnerabilities']},
      },
    ],
  },
  {
    id: 'configuration',
    label: 'Configuration',
    entries: [
      {
        id: 'customer-portal',
        label: 'Customer Portal',
        icon: faPalette,
        route: '/customer-portal',
        shortcut: 'o',
        visible: (c) => c.user.isSuperAdmin || c.user.isReadWriteOrAdmin,
      },
      {id: 'settings', label: 'Settings', icon: faGear, route: '/settings/organization', shortcut: 's'},
    ],
  },
];
