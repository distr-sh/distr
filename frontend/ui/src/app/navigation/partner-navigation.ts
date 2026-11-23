import {
  faAddressBook,
  faBox,
  faBoxesStacked,
  faHome,
  faLifeRing,
  faShieldHalved,
  faUsers,
} from '@fortawesome/free-solid-svg-icons';
import {NavigationDefinition} from './navigation';

export const partnerNavigation: NavigationDefinition = [
  {
    id: 'distribution',
    label: 'Distribution',
    entries: [
      {id: 'home', label: 'Home', icon: faHome, route: '/home', shortcut: 'h'},
      {id: 'deployments', label: 'Deployments', icon: faBoxesStacked, route: '/deployments', shortcut: 'd'},
      {id: 'registry', label: 'Registry', icon: faBox, route: '/registry', shortcut: 'r'},
      {
        id: 'support',
        label: 'Support',
        icon: faLifeRing,
        route: '/support',
        shortcut: 'b',
        visible: (c) => c.features.has('support'),
      },
      {
        id: 'security',
        label: 'Security',
        icon: faShieldHalved,
        route: '/security',
        shortcut: 'v',
        visible: (c) => c.features.has('vulnerabilities'),
      },
    ],
  },
  {
    id: 'management',
    label: 'Management',
    entries: [
      {id: 'customers', label: 'Customers', icon: faAddressBook, route: '/customers', shortcut: 'c'},
      {
        id: 'team',
        label: 'Team',
        icon: faUsers,
        route: '/team',
        shortcut: 't',
        visible: (c) => c.user.isAdmin,
      },
    ],
  },
];
