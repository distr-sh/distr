import type {MegaMenuType} from '~/data/megaMenuConfig';

export type NavigationLink = {
  title: string;
  value: string;
  children?: NavigationLink[];
  isMegaMenu?: boolean;
  megaMenuType?: MegaMenuType;
};

export const links: NavigationLink[] = [
  {
    title: 'Product',
    value: '/docs/product/agents/',
    isMegaMenu: true,
    megaMenuType: 'products',
  },
  {
    title: 'Docs',
    value: '/docs/',
    isMegaMenu: true,
    megaMenuType: 'docs',
  },
  {
    title: 'Pricing',
    value: '/pricing/',
    isMegaMenu: true,
    megaMenuType: 'pricing',
  },
  {
    title: 'Resources',
    value: '/blog/',
    isMegaMenu: true,
    megaMenuType: 'resources',
  },
];
