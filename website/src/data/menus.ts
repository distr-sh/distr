export type MenuItem = {
  title: string;
  description: string;
  value: string;
  href: string;
};

export const docsMenu: MenuItem[] = [
  {
    title: 'Docs',
    description:
      'Complete documentation for getting started with Distr and understanding core concepts',
    value: 'book-open',
    href: '/docs/',
  },
  {
    title: 'Guides',
    description: 'Step-by-step guides for common tasks and workflows in Distr',
    value: 'map',
    href: '/docs/guides/',
  },
  {
    title: 'Integrations',
    description:
      'Connect Distr with your existing tools and workflows through our API, SDK, and integrations',
    value: 'plug',
    href: '/docs/integrations/',
  },
  {
    title: 'Self-Hosting',
    description:
      'Deploy and manage your own Distr instance on Kubernetes or Docker',
    value: 'server',
    href: '/docs/self-hosting/',
  },
];

export const pricingMenu: MenuItem[] = [
  {
    title: 'Pricing',
    description:
      'Flexible pricing plans for teams of all sizes, from startups to enterprises',
    value: 'credit-card',
    href: '/pricing/',
  },
  {
    title: 'Contact',
    description:
      'Get in touch with our team for custom solutions and enterprise support',
    value: 'mail',
    href: '/contact/',
  },
];

export const resourcesMenu: MenuItem[] = [
  {
    title: 'Blog',
    description: 'Latest news, updates, and insights from the Distr team',
    value: 'newspaper',
    href: '/blog/',
  },
  {
    title: 'Case Studies',
    description:
      'Learn how companies are using Distr to distribute their software',
    value: 'briefcase',
    href: '/case-studies/',
  },
  {
    title: 'Glossary',
    description: 'Understand key terms and concepts in software distribution',
    value: 'book-text',
    href: '/glossary/',
  },
  {
    title: 'White Paper',
    description:
      'Deep dive into the building blocks of modern software distribution',
    value: 'file-text',
    href: '/white-paper/building-blocks/',
  },
];
