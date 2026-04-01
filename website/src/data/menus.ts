export type MenuItem = {
  title: string;
  description: string;
  value: string;
  href: string;
};

export const docsMenu: MenuItem[] = [
  {
    title: 'Getting Started',
    description:
      'Introduction, core concepts, quickstart, and distribution scenarios',
    value: 'book-open',
    href: '/docs/',
  },
  {
    title: 'Deployment Agents',
    description:
      'Set up and configure Docker and Helm agents for your deployments',
    value: 'rocket',
    href: '/docs/agents/',
  },
  {
    title: 'Artifact Registry',
    description:
      'Distribute Docker images, Helm charts, and OCI artifacts with the built-in registry',
    value: 'package',
    href: '/docs/registry/',
  },
  {
    title: 'Distribution Platform',
    description:
      'License management, entitlements, customer portal, and compliance tools',
    value: 'layout-grid',
    href: '/docs/platform/license-management/',
  },
  {
    title: 'Integrations & API',
    description:
      'Connect Distr with GitHub Actions, the REST API, and the TypeScript SDK',
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
    title: 'White Paper',
    description:
      'Deep dive into the building blocks of modern software distribution',
    value: 'file-text',
    href: '/white-paper/building-blocks/',
  },
];
