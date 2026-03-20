export type Product = {
  title: string;
  description: string;
  value: string;
  href: string;
};

export const products: Product[] = [
  {
    title: 'Deployment Agents',
    description:
      'Standardize your Docker or Helm deployments embedded with agents in Kubernetes, Docker Compose, VM or Bare Metal environments',
    value: 'rocket',
    href: '/docs/product/agents/',
  },
  {
    title: 'Customer Portal',
    description:
      'Where customers can download Artifacts and manage their Applications',
    value: 'circle-user',
    href: '/docs/product/customer-portal/',
  },
  {
    title: 'Container Registry',
    description:
      'Distribute Docker Images, Helm Charts, or every OCI compatible artifact like large LLM Models with our high performance OCI registry',
    value: 'package',
    href: '/docs/product/registry/',
  },
  {
    title: 'License Management',
    description:
      'Time based access control and entitlements for Applications and Artifacts - managed in a central place',
    value: 'key',
    href: '/docs/product/license-management/',
  },
  {
    title: 'Alerts',
    description:
      'Receive notification in realtime when deployments report an error or go stale',
    value: 'bell',
    href: '/docs/product/alerts/',
  },
  {
    title: 'Logs and Metrics',
    description:
      'Collect Logs and Metrics in realtime and directly download them as bundle for better Customer support',
    value: 'chart-line',
    href: '/docs/product/agents/#container-logs',
  },
  {
    title: 'Compatibility Matrix',
    description:
      'New - Automatically test if your application is compatible across a matrix of deployment environments',
    value: 'check-circle',
    href: '/docs/guides/kubernetes-compatibility-matrix/',
  },
  {
    title: 'Pre-flight Checks',
    description:
      'New - Determine if needed resources are available or execute custom pre and post installation scripts',
    value: 'clipboard-check',
    href: '/docs/guides/preflight-checks/',
  },
  {
    title: 'Vulnerability Scanning',
    description:
      'New - Scan source dependencies and container images for known vulnerabilities and share reports with customers',
    value: 'shield-check',
    href: '/docs/guides/vulnerability-scanning/',
  },
  {
    title: 'Secrets',
    description:
      'Securely store and reference sensitive configuration values in deployments using centralized secret management',
    value: 'lock',
    href: '/docs/guides/secrets/',
  },
  {
    title: 'Integrations / BYOC',
    description:
      'Extend your pull based deployment model with push based approaches with our API Integrations and GitHub Action',
    value: 'workflow',
    href: '/docs/guides/automatic-deployments-from-github/',
  },
  {
    title: 'Air-gapped',
    description:
      'Enterprise - Distribute your application with air-gapped bundles into the most isolated environments',
    value: 'server-off',
    href: '/docs/use-cases/air-gapped/',
  },
];
