import {faBox} from '@fortawesome/free-solid-svg-icons';
import {NavigationGroupDefinition} from '../navigation';

export const registryGroup: NavigationGroupDefinition = {
  id: 'registry',
  label: 'Registry',
  icon: faBox,
  children: [
    {id: 'artifacts', label: 'Artifacts', route: '/artifacts'},
    {id: 'artifact-pulls', label: 'Downloads', route: '/artifact-pulls'},
  ],
};
