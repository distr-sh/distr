import {
  faAddressBook,
  faBox,
  faBoxesStacked,
  faHome,
  faKey,
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
      {id: 'home', label: 'Home', icon: faHome, route: '/home'},
      {id: 'deployments', label: 'Deployments', icon: faBoxesStacked, route: '/deployments'},
      {id: 'registry', label: 'Registry', icon: faBox, route: '/artifacts'},
      {
        id: 'licenses',
        label: 'Licenses',
        icon: faKey,
        route: '/licenses',
        visible: (c) => c.features.has('licensing'),
      },
      {
        id: 'support',
        label: 'Support',
        icon: faLifeRing,
        route: '/support',
        visible: (c) => c.features.has('support_bundles'),
      },
      {
        id: 'advisories',
        label: 'Security',
        icon: faShieldHalved,
        route: '/advisories',
        visible: (c) => c.features.has('vulnerabilities'),
      },
    ],
  },
  {
    id: 'management',
    label: 'Management',
    entries: [
      {id: 'customers', label: 'Customers', icon: faAddressBook, route: '/customers'},
      {id: 'users', label: 'Users', icon: faUsers, route: '/users', visible: (c) => c.user.isAdmin},
    ],
  },
];
