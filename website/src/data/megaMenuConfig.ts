import type {MenuItem} from '~/data/menus';
import {docsMenu, pricingMenu, resourcesMenu} from '~/data/menus';
import type {Product} from '~/data/products';
import {products} from '~/data/products';

export type MegaMenuType = 'products' | 'docs' | 'pricing' | 'resources';

export interface MegaMenuConfig {
  items: MenuItem[] | Product[];
  columns: number;
  itemWidth: string;
}

export const megaMenuConfigs: Record<MegaMenuType, MegaMenuConfig> = {
  products: {
    items: products,
    columns: 5,
    itemWidth: '140px',
  },
  docs: {
    items: docsMenu,
    columns: 4,
    itemWidth: '160px',
  },
  pricing: {
    items: pricingMenu,
    columns: 2,
    itemWidth: '200px',
  },
  resources: {
    items: resourcesMenu,
    columns: 4,
    itemWidth: '160px',
  },
};
