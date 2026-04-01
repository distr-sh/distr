// @ts-check
import mdx from '@astrojs/mdx';
import preact from '@astrojs/preact';
import sitemap from '@astrojs/sitemap';
import starlight from '@astrojs/starlight';
import tailwindcss from '@tailwindcss/vite';
import icon from 'astro-icon';
import {defineConfig, fontProviders} from 'astro/config';
import serviceWorker from 'astrojs-service-worker';
import rehypeMermaid from 'rehype-mermaid';
import starlightLinksValidator from 'starlight-links-validator';
import starlightSidebarTopics from 'starlight-sidebar-topics';

// https://astro.build/config
export default defineConfig({
  site: 'https://distr.sh',
  fonts: [
    {
      name: 'Inter',
      cssVariable: '--font-inter',
      provider: fontProviders.fontsource(),
      weights: [300, 400, 600, 700],
      subsets: ['latin'],
    },
    {
      name: 'Poppins',
      cssVariable: '--font-poppins',
      provider: fontProviders.fontsource(),
      weights: [600],
      subsets: ['latin'],
    },
  ],

  integrations: [
    icon({include: {lucide: ['*']}}),
    preact(),
    sitemap({
      filter: page => {
        // Exclude specific pages by slug
        const excludedSlugs = [
          '/onboarding/',
          '/get-started/',
          '/docs/',
          '/demo/success/',
        ];
        const url = new URL(page);
        const pathname = url.pathname;

        return !excludedSlugs.some(slug => slug === pathname);
      },
    }),
    serviceWorker(),
    starlight({
      title: 'Distr',
      customCss: ['./src/styles/global.css'],
      editLink: {
        baseUrl: 'https://github.com/distr-sh/distr/tree/main/website',
      },
      lastUpdated: true,
      head:
        process.env.NODE_ENV === 'production'
          ? [
              {
                tag: 'script',
                attrs: {
                  type: 'text/javascript',
                },
                content: `(function(w,d,s,l,i){w[l]=w[l]||[];w[l].push({'gtm.start':
new Date().getTime(),event:'gtm.js'});var f=d.getElementsByTagName(s)[0],
j=d.createElement(s),dl=l!='dataLayer'?'&l='+l:'';j.async=true;j.src=
'https://www.googletagmanager.com/gtm.js?id='+i+dl;f.parentNode.insertBefore(j,f);
})(window,document,'script','dataLayer','GTM-T58STPCJ');`,
              },
            ]
          : [],
      description: 'Open Source Software Distribution Platform',
      logo: {
        src: './src/assets/distr.svg',
      },
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/distr-sh/distr',
        },
        {
          icon: 'discord',
          label: 'Discord',
          href: 'https://discord.gg/6qqBSAWZfW',
        },
      ],
      components: {
        // Components can be overwritten here
        Head: './src/components/overwrites/Head.astro',
        Header: './src/components/overwrites/Header.astro',
        PageTitle: './src/components/overwrites/PageTitle.astro',
        ContentPanel: './src/components/overwrites/ContentPanel.astro',
        Footer: './src/components/overwrites/Footer.astro',
        SocialIcons: './src/components/overwrites/SocialIcons.astro',
        ThemeProvider: './src/components/overwrites/ThemeProvider.astro',
        ThemeSelect: './src/components/overwrites/ThemeSelect.astro',
      },
      tableOfContents: {
        minHeadingLevel: 2,
        maxHeadingLevel: 6,
      },
      prerender: true,
      plugins: [
        starlightSidebarTopics(
          [
            {
              label: 'Getting Started',
              link: '/docs/',
              icon: 'open-book',
              items: [
                {
                  label: 'Introduction',
                  items: [
                    {label: 'What is Distr?', link: '/docs/'},
                    {label: 'Core Concepts', link: '/docs/concepts/'},
                    {
                      label: 'Vendor Portal',
                      link: '/docs/platform/vendor-portal/',
                    },
                    {label: 'Quickstart', link: '/docs/quickstart/'},
                    {label: 'FAQs', link: '/docs/faqs/'},
                  ],
                },
                {
                  label: 'Distribution Scenarios',
                  items: [
                    {
                      label: 'Fully Self-Managed',
                      link: '/docs/use-cases/fully-self-managed/',
                    },
                    {
                      label: 'Assisted Self-Managed',
                      link: '/docs/use-cases/assisted-self-managed/',
                    },
                    {label: 'BYOC', link: '/docs/use-cases/byoc/'},
                    {
                      label: 'Air-Gapped Deployments',
                      link: '/docs/use-cases/air-gapped/',
                    },
                    {
                      label: 'Edge Deployments',
                      link: '/docs/use-cases/edge/',
                    },
                  ],
                },
                {
                  label: 'Account',
                  items: [
                    {label: 'Free Trial', link: '/docs/account/trial/'},
                    {label: 'Choosing a Plan', link: '/docs/account/plans/'},
                    {
                      label: 'Subscription Management',
                      link: '/docs/account/subscription/',
                    },
                  ],
                },
              ],
            },
            {
              label: 'Deployment Agents',
              link: '/docs/agents/',
              icon: 'random',
              items: [
                {
                  label: 'Overview & Setup',
                  items: [
                    {
                      label: 'Deployment Agents',
                      link: '/docs/agents/',
                    },
                    {
                      label: 'Docker Agent',
                      link: '/docs/agents/docker-agent/',
                    },
                    {
                      label: 'Helm Agent',
                      link: '/docs/agents/helm-agent/',
                    },
                    {
                      label: 'Create an Application',
                      link: '/docs/agents/create-application/',
                    },
                    {
                      label: 'Create a Deployment',
                      link: '/docs/agents/create-deployment/',
                    },
                    {
                      label: 'Run on macOS',
                      link: '/docs/agents/setup-on-macos/',
                    },
                  ],
                },
                {
                  label: 'Configuration',
                  items: [
                    {
                      label: 'Configure Docker Env Variables',
                      link: '/docs/agents/docker-env/',
                    },
                    {
                      label: 'Secrets Management',
                      link: '/docs/agents/secrets/',
                    },
                    {
                      label: 'Docker Compose Secrets',
                      link: '/docs/agents/docker-compose-secrets/',
                    },
                    {
                      label: 'Configure Application Links',
                      link: '/docs/agents/application-links/',
                    },
                    {
                      label: 'Configuring Helm Charts for Distr Artifacts',
                      link: '/docs/agents/helm-registry-auth/',
                    },
                  ],
                },
                {
                  label: 'Monitoring',
                  items: [
                    {label: 'Alerts', link: '/docs/agents/alerts/'},
                    {
                      label: 'Logs & Metrics',
                      link: '/docs/agents/logs-and-metrics/',
                    },
                    {
                      label: 'Pre-flight Checks',
                      link: '/docs/agents/preflight-checks/',
                    },
                  ],
                },
              ],
            },
            {
              label: 'Container Registry',
              link: '/docs/registry/',
              icon: 'download',
              items: [
                {
                  label: 'Artifact Registry',
                  link: '/docs/registry/',
                },
                {
                  label: 'Set Up Your Registry',
                  link: '/docs/registry/push-to-registry/',
                },
                {
                  label: 'Download Analytics',
                  link: '/docs/registry/download-analytics/',
                },
              ],
            },
            {
              label: 'Distribution Platform',
              link: '/docs/platform/license-management/',
              icon: 'list-format',
              items: [
                {
                  label: 'License Management',
                  items: [
                    {
                      label: 'License Management Overview',
                      link: '/docs/platform/license-management/',
                    },
                    {
                      label: 'Application Entitlements',
                      link: '/docs/platform/application-entitlements/',
                    },
                    {
                      label: 'Artifact Entitlements',
                      link: '/docs/platform/artifact-entitlements/',
                    },
                    {
                      label: 'License Keys',
                      link: '/docs/platform/license-keys/',
                    },
                  ],
                },
                {
                  label: 'Quality & Compliance',
                  items: [
                    {
                      label: 'Support Bundles',
                      link: '/docs/platform/support-bundles/',
                    },
                    {
                      label: 'Kubernetes Compatibility Matrix',
                      link: '/docs/platform/k8s-compatibility/',
                    },
                    {
                      label: 'Vulnerability Scanning',
                      link: '/docs/platform/vulnerability-scanning/',
                    },
                  ],
                },
                {
                  label: 'Customer Portal',
                  items: [
                    {
                      label: 'Customer Portal',
                      link: '/docs/customers/',
                    },
                    {
                      label: 'End-Customer View of Distr',
                      link: '/docs/customers/customer-registry-access/',
                    },
                  ],
                },
                {
                  label: 'Customer Management',
                  items: [
                    {
                      label: 'Branding & White-Labeling',
                      link: '/docs/customers/branding/',
                    },
                    {
                      label: 'Role-Based Access Control (RBAC)',
                      link: '/docs/customers/rbac/',
                    },
                    {
                      label: 'Manage Customers',
                      link: '/docs/customers/manage-customers/',
                    },
                  ],
                },
              ],
            },
            {
              label: 'Integrations & API',
              link: '/docs/integrations/github-actions/',
              icon: 'rocket',
              items: [
                {
                  label: 'GitHub Actions',
                  items: [
                    {
                      label: 'Automatic Deployments from GitHub',
                      link: '/docs/integrations/github-actions/',
                    },
                    {
                      label: 'GitHub Action Reference',
                      link: '/docs/integrations/github-action/',
                    },
                  ],
                },
                {
                  label: 'API & SDK',
                  items: [
                    {label: 'Distr API', link: '/docs/integrations/rest-api/'},
                    {
                      label: 'API Reference',
                      link: 'https://app.distr.sh/docs',
                      attrs: {target: '_blank'},
                    },
                    {label: 'Distr SDK', link: '/docs/integrations/sdk/'},
                    {
                      label: 'Personal Access Tokens',
                      link: '/docs/integrations/access-tokens/',
                    },
                  ],
                },
              ],
            },
            {
              label: 'Self-Hosting',
              link: '/docs/self-hosting/',
              icon: 'laptop',
              items: [
                {
                  label: 'Self-Hosting',
                  autogenerate: {directory: 'docs/self-hosting'},
                },
              ],
            },
          ],
          {
            exclude: ['**/privacy-policy', '**/404'],
          },
        ),
        starlightLinksValidator({
          exclude: [
            '/',
            '/contact/',
            '/pricing/',
            '/blog/**',
            '/glossary/**',
            '/get-started/',
            '/onboarding/',
            'mailto:**',
          ],
        }),
      ],
    }),
    mdx(),
  ],
  markdown: {
    rehypePlugins: [[rehypeMermaid, {strategy: 'inline-svg'}]],
  },
  vite: {
    plugins: [tailwindcss()],
  },
  redirects: {
    // Legacy deep-link redirects
    '/docs/getting-started/': '/docs/',
    '/docs/getting-started/about/': '/docs/',
    '/docs/getting-started/what-is-distr/': '/docs/',
    '/docs/getting-started/how-it-works/': '/docs/concepts/',
    '/docs/getting-started/core-concepts/': '/docs/concepts/',
    '/docs/getting-started/quickstart/': '/docs/quickstart/',
    '/docs/getting-started/deployment-methods/': '/docs/account/plans/',
    '/docs/privacy-policy/': '/privacy-policy/',

    // intro/ paths
    '/docs/intro/about/': '/docs/',
    '/docs/intro/core-concepts/': '/docs/concepts/',
    '/docs/intro/quickstart/': '/docs/quickstart/',
    '/docs/intro/free-trial/': '/docs/account/trial/',
    '/docs/intro/subscription/': '/docs/account/plans/',

    // product/ → platform/
    '/docs/product/vendor-portal/': '/docs/platform/vendor-portal/',
    '/docs/product/agents/': '/docs/agents/',
    '/docs/product/alerts/': '/docs/agents/alerts/',
    '/docs/product/registry/': '/docs/registry/',
    '/docs/product/support-bundles/': '/docs/platform/support-bundles/',
    '/docs/product/customer-portal/': '/docs/customers/',
    '/docs/product/branding/': '/docs/customers/branding/',
    '/docs/product/rbac/': '/docs/customers/rbac/',
    '/docs/product/license-management/': '/docs/platform/license-management/',
    '/docs/product/subscription-management/': '/docs/account/subscription/',
    '/docs/product/distr-hub/': '/docs/platform/vendor-portal/',
    '/docs/product/faqs/': '/docs/faqs/',

    // use-cases
    '/docs/use-cases/self-managed/': '/docs/use-cases/fully-self-managed/',
    '/docs/use-cases/edge-deployments/': '/docs/use-cases/edge/',
    '/docs/use-cases/air-gapped-deployments/': '/docs/use-cases/air-gapped/',
    '/docs/use-cases/byoc-bring-your-own-cloud/': '/docs/use-cases/byoc/',

    // Guide renames
    '/docs/guides/getting-started/application/':
      '/docs/agents/create-application/',
    '/docs/guides/getting-started/deployment/':
      '/docs/agents/create-deployment/',
    '/docs/guides/getting-started/distr-on-macos/':
      '/docs/agents/setup-on-macos/',
    '/docs/guides/getting-started/how-to-registry/':
      '/docs/registry/push-to-registry/',
    '/docs/guides/configuration/docker-env/': '/docs/agents/docker-env/',
    '/docs/guides/configuration/docker-secrets/':
      '/docs/agents/docker-compose-secrets/',
    '/docs/guides/advanced/secrets/': '/docs/agents/secrets/',
    '/docs/guides/configuration/application-links/':
      '/docs/agents/application-links/',
    '/docs/guides/configuration/helm-chart-registry-auth/':
      '/docs/agents/helm-registry-auth/',
    '/docs/guides/automation/automatic-deployments-from-github/':
      '/docs/integrations/github-actions/',
    '/docs/guides/automation/preflight-checks/':
      '/docs/agents/preflight-checks/',
    '/docs/guides/automation/kubernetes-compatibility-matrix/':
      '/docs/platform/k8s-compatibility/',
    '/docs/guides/automation/vulnerability-scanning/':
      '/docs/platform/vulnerability-scanning/',
    '/docs/guides/customer-management/customer-management/':
      '/docs/customers/manage-customers/',
    '/docs/guides/customer-management/end-customer-registry-view/':
      '/docs/customers/customer-registry-access/',
    '/docs/guides/customer-management/application-entitlements/':
      '/docs/platform/application-entitlements/',
    '/docs/guides/customer-management/artifact-entitlements/':
      '/docs/platform/artifact-entitlements/',
    '/docs/guides/customer-management/license-keys/':
      '/docs/platform/license-keys/',

    // Legacy guide slugs
    '/docs/guides/license-mgmt/': '/docs/platform/application-entitlements/',
    '/docs/guides/application-licenses/':
      '/docs/platform/application-entitlements/',
    '/docs/guides/artifact-licenses/': '/docs/platform/artifact-entitlements/',
    '/docs/guides/onboarding-a-new-customer/': '/docs/customers/rbac/',
    '/docs/guides/onboarding-a-docker-app/': '/docs/agents/create-application/',
    '/docs/guides/onboarding-a-helm-app/': '/docs/agents/create-application/',

    // Integration renames
    '/docs/integrations/mcp/': '/docs/',
    '/docs/integrations/api/': '/docs/integrations/rest-api/',
    '/docs/integrations/gh-action/': '/docs/integrations/github-action/',
    '/docs/integrations/personal-access-token/':
      '/docs/integrations/access-tokens/',

    // Deleted index pages
    '/docs/integrations/': '/docs/integrations/rest-api/',
    '/docs/guides/': '/docs/quickstart/',

    // New URL restructure redirects
    '/docs/platform/agents/': '/docs/agents/',
    '/docs/platform/registry/': '/docs/registry/',
    '/docs/platform/customer-portal/': '/docs/customers/',
    '/docs/platform/alerts/': '/docs/agents/alerts/',
    '/docs/platform/logs-and-metrics/': '/docs/agents/logs-and-metrics/',
    '/docs/platform/branding/': '/docs/customers/branding/',
    '/docs/platform/rbac/': '/docs/customers/rbac/',
    '/docs/guides/create-application/': '/docs/agents/create-application/',
    '/docs/guides/create-deployment/': '/docs/agents/create-deployment/',
    '/docs/guides/setup-on-macos/': '/docs/agents/setup-on-macos/',
    '/docs/guides/docker-env/': '/docs/agents/docker-env/',
    '/docs/guides/secrets/': '/docs/agents/secrets/',
    '/docs/guides/application-links/': '/docs/agents/application-links/',
    '/docs/guides/helm-registry-auth/': '/docs/agents/helm-registry-auth/',
    '/docs/guides/preflight-checks/': '/docs/agents/preflight-checks/',
    '/docs/guides/push-to-registry/': '/docs/registry/push-to-registry/',
    '/docs/guides/application-entitlements/':
      '/docs/platform/application-entitlements/',
    '/docs/guides/artifact-entitlements/':
      '/docs/platform/artifact-entitlements/',
    '/docs/guides/license-keys/': '/docs/platform/license-keys/',
    '/docs/guides/k8s-compatibility/': '/docs/platform/k8s-compatibility/',
    '/docs/guides/vulnerability-scanning/':
      '/docs/platform/vulnerability-scanning/',
    '/docs/guides/manage-customers/': '/docs/customers/manage-customers/',
    '/docs/guides/customer-registry-access/':
      '/docs/customers/customer-registry-access/',
    '/docs/guides/github-actions/': '/docs/integrations/github-actions/',
    '/docs/customers/subscription/': '/docs/account/subscription/',
    '/docs/platform/subscription/': '/docs/account/subscription/',
  },
});
