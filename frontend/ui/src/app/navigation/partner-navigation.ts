import {
  faAddressBook,
  faBox,
  faBoxesStacked,
  faDashboard,
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
      {id: 'dashboard', label: 'Dashboard', icon: faDashboard, route: '/dashboard'},
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
        visible: (c) => c.features.has('licensing'),
      },
      {
        id: 'support-bundles',
        label: 'Support Bundles',
        icon: faLifeRing,
        route: '/support-bundles',
        visible: (c) => c.features.has('support_bundles'),
      },
      {
        id: 'advisories',
        label: 'Security Advisories',
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
