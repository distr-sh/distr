export type NavigationLink = {
  title: string;
  value: string;
  children?: NavigationLink[];
};

export const links: NavigationLink[] = [
  {
    title: 'Docs',
    value: '/docs/',
  },
  {
    title: 'Pricing',
    value: '/pricing/',
  },
  {
    title: 'Resources',
    value: 'javascript:void(0)',
    children: [
      {
        title: 'Blog',
        value: '/blog/',
      },
      {
        title: 'Case Studies',
        value: '/case-studies/',
      },
      {
        title: 'Glossary',
        value: '/glossary/',
      },
      {
        title: 'White Paper',
        value: '/white-paper/building-blocks/',
      },
    ],
  },
];
