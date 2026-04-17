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
                    {label: 'Core Concepts', link: '/docs/core-concepts/'},
                    {
                      label: 'Vendor Portal',
                      link: '/docs/vendor-portal/',
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
                    {
                      label: 'BYOC',
                      link: '/docs/use-cases/byoc-bring-your-own-cloud/',
                    },
                    {
                      label: 'Air-Gapped Deployments',
                      link: '/docs/use-cases/air-gapped-deployments/',
                    },
                    {
                      label: 'Edge Deployments',
                      link: '/docs/use-cases/edge-deployments/',
                    },
                  ],
                },
                {
                  label: 'Account',
                  items: [
                    {label: 'Free Trial', link: '/docs/free-trial/'},
                    {label: 'Choosing a Plan', link: '/docs/subscription/'},
                    {
                      label: 'Subscription Management',
                      link: '/docs/subscription-management/',
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
                      label: 'Kubernetes Agent',
                      link: '/docs/agents/kubernetes-agent/',
                    },
                    {
                      label: 'Application',
                      link: '/docs/agents/application/',
                    },
                    {
                      label: 'Deployment',
                      link: '/docs/agents/deployment/',
                    },
                    {
                      label: 'Run on macOS',
                      link: '/docs/agents/distr-on-macos/',
                    },
                  ],
                },
                {
                  label: 'Configuration',
                  items: [
                    {
                      label: 'Docker Environment Variables',
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
                      label: 'Application Links',
                      link: '/docs/agents/application-links/',
                    },
                    {
                      label: 'Helm Chart Registry Authentication',
                      link: '/docs/agents/helm-registry-auth/',
                    },
                    {
                      label: 'Pre-Flight Checks',
                      link: '/docs/agents/preflight-checks/',
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
                  ],
                },
              ],
            },
            {
              label: 'Artifact Registry',
              link: '/docs/registry/',
              icon: 'download',
              items: [
                {
                  label: 'Overview',
                  items: [
                    {
                      label: 'Artifact Registry',
                      link: '/docs/registry/',
                    },
                    {
                      label: 'Registry Configuration',
                      link: '/docs/registry/configuration/',
                    },
                    {
                      label: 'Artifact Download Analytics',
                      link: '/docs/registry/analytics/',
                    },
                  ],
                },
              ],
            },
            {
              label: 'Distribution Platform',
              link: '/docs/platform/',
              icon: 'list-format',
              items: [
                {
                  label: 'License Management',
                  items: [
                    {
                      label: 'Overview',
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
                  label: 'Support',
                  items: [
                    {
                      label: 'Support Bundles',
                      link: '/docs/platform/support-bundles/',
                    },
                    {
                      label: 'Kubernetes Compatibility Matrix',
                      link: '/docs/platform/kubernetes-compatibility-matrix/',
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
                      label: 'Overview',
                      link: '/docs/platform/customer-portal/',
                    },
                    {
                      label: 'Artifact Registry',
                      link: '/docs/platform/customer-portal/registry/',
                    },
                    {
                      label: 'Deployments',
                      link: '/docs/platform/customer-portal/deployments/',
                    },
                    {
                      label: 'Licenses',
                      link: '/docs/platform/customer-portal/licenses/',
                    },
                    {
                      label: 'Support Bundles',
                      link: '/docs/platform/customer-portal/support/',
                    },
                    {
                      label: 'Secrets',
                      link: '/docs/platform/customer-portal/secrets/',
                    },
                  ],
                },
                {
                  label: 'Customer Management',
                  items: [
                    {
                      label: 'Overview',
                      link: '/docs/platform/customer-management/',
                    },
                    {
                      label: 'Branding & White-Labeling',
                      link: '/docs/platform/branding/',
                    },
                  ],
                },
                {
                  label: 'User Management',
                  items: [
                    {
                      label: 'Role-Based Access Control (RBAC)',
                      link: '/docs/platform/rbac/',
                    },
                  ],
                },
              ],
            },
            {
              label: 'Integrations & API',
              link: '/docs/integrations/',
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
                      link: '/docs/integrations/gh-action/',
                    },
                  ],
                },
                {
                  label: 'API & SDK',
                  items: [
                    {label: 'Distr API', link: '/docs/integrations/api/'},
                    {
                      label: 'API Reference',
                      link: 'https://app.distr.sh/docs',
                      attrs: {target: '_blank'},
                    },
                    {label: 'Distr SDK', link: '/docs/integrations/sdk/'},
                    {
                      label: 'Personal Access Tokens',
                      link: '/docs/integrations/personal-access-token/',
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
    '/docs/getting-started/how-it-works/': '/docs/core-concepts/',
    '/docs/getting-started/core-concepts/': '/docs/core-concepts/',
    '/docs/getting-started/quickstart/': '/docs/quickstart/',
    '/docs/getting-started/deployment-methods/': '/docs/subscription/',
    '/docs/privacy-policy/': '/privacy-policy/',

    // product/ redirects
    '/docs/product/vendor-portal/': '/docs/vendor-portal/',
    '/docs/product/agents/': '/docs/agents/',
    '/docs/product/alerts/': '/docs/agents/alerts/',
    '/docs/product/registry/': '/docs/registry/',
    '/docs/product/support-bundles/': '/docs/platform/support-bundles/',
    '/docs/product/customer-portal/': '/docs/platform/customer-portal/',
    '/docs/product/branding/': '/docs/platform/branding/',
    '/docs/product/rbac/': '/docs/platform/rbac/',
    '/docs/product/license-management/': '/docs/platform/license-management/',
    '/docs/product/subscription-management/': '/docs/subscription-management/',
    '/docs/product/distr-hub/': '/docs/vendor-portal/',
    '/docs/product/faqs/': '/docs/faqs/',

    // use-cases
    '/docs/use-cases/self-managed/': '/docs/use-cases/fully-self-managed/',
    '/docs/use-cases/byoc/': '/docs/use-cases/byoc-bring-your-own-cloud/',
    '/docs/use-cases/air-gapped/': '/docs/use-cases/air-gapped-deployments/',

    // guides/ redirects (slugs that existed on main)
    '/docs/guides/': '/docs/quickstart/',
    '/docs/guides/secrets/': '/docs/agents/secrets/',
    '/docs/guides/application-links/': '/docs/agents/application-links/',
    '/docs/guides/preflight-checks/': '/docs/agents/preflight-checks/',
    '/docs/guides/application-entitlements/':
      '/docs/platform/application-entitlements/',
    '/docs/guides/artifact-entitlements/':
      '/docs/platform/artifact-entitlements/',
    '/docs/guides/license-keys/': '/docs/platform/license-keys/',
    '/docs/guides/vulnerability-scanning/':
      '/docs/platform/vulnerability-scanning/',
    '/docs/guides/container-registry/': '/docs/registry/configuration/',
    '/docs/guides/docker-secrets/': '/docs/agents/docker-compose-secrets/',
    '/docs/guides/container-registry-for-end-customers/':
      '/docs/platform/customer-portal/registry/',
    '/docs/guides/license-mgmt/': '/docs/platform/application-entitlements/',
    '/docs/guides/application-licenses/':
      '/docs/platform/application-entitlements/',
    '/docs/guides/artifact-licenses/': '/docs/platform/artifact-entitlements/',
    '/docs/guides/onboarding-a-new-customer/': '/docs/platform/rbac/',
    '/docs/guides/onboarding-a-docker-app/': '/docs/agents/application/',
    '/docs/guides/onboarding-a-helm-app/': '/docs/agents/application/',

    // Integration redirects
    '/docs/integrations/mcp/': '/docs/integrations/',

    // Legacy blog redirects (content superseded by /compare/ pages)
    '/blog/distr-vs-replicated/': '/compare/replicated/',
  },
});
