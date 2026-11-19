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
  faLightbulb,
  faPalette,
  faShieldHalved,
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
      {
        id: 'partners',
        label: 'Partners',
        icon: faBuilding,
        route: '/partners',
        visible: (c) => c.features.has('partner_management'),
      },
      {id: 'customers', label: 'Customers', icon: faAddressBook, route: '/customers'},
      {id: 'deployments', label: 'Deployments', icon: faBoxesStacked, route: '/deployments'},
      {id: 'registry', label: 'Registry', icon: faBox, route: '/registry'},
      {
        id: 'notifications',
        label: 'Notifications',
        icon: faBell,
        route: '/notifications',
        locked: {requiresAnyOf: ['notifications']},
      },
      {
        id: 'support',
        label: 'Support',
        icon: faLifeRing,
        route: '/support',
        locked: {requiresAnyOf: ['support']},
      },
      {
        id: 'security',
        label: 'Security',
        icon: faShieldHalved,
        route: '/security',
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
        visible: (c) => c.user.isSuperAdmin || c.user.isReadWriteOrAdmin,
      },
      {id: 'settings', label: 'Settings', icon: faGear, route: '/settings/organization'},
    ],
  },
];
