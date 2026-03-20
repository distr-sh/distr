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
                {label: 'What is Distr?', link: '/docs/'},
                {label: 'Vendor Portal', link: '/docs/platform/vendor-portal/'},
                {label: 'Core Concepts', link: '/docs/concepts/'},
                {label: 'Quickstart', link: '/docs/quickstart/'},
                {label: 'Free Trial', link: '/docs/account/trial/'},
                {label: 'Choosing a Plan', link: '/docs/account/plans/'},
                {label: 'FAQs', link: '/docs/faqs/'},
              ],
            },
            {
              label: 'Distribution Scenarios',
              link: '/docs/use-cases/fully-self-managed/',
              icon: 'puzzle',
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
                {label: 'Edge Deployments', link: '/docs/use-cases/edge/'},
              ],
            },
            {
              label: 'Deployment Agents',
              link: '/docs/platform/agents/',
              icon: 'setting',
              items: [
                {
                  label: 'Overview & Setup',
                  items: [
                    {
                      label: 'Deployment Agents',
                      link: '/docs/platform/agents/',
                    },
                    {
                      label: 'Create an Application',
                      link: '/docs/guides/create-application/',
                    },
                    {
                      label: 'Create a Deployment',
                      link: '/docs/guides/create-deployment/',
                    },
                    {
                      label: 'Run on macOS',
                      link: '/docs/guides/setup-on-macos/',
                    },
                  ],
                },
                {
                  label: 'Configuration',
                  items: [
                    {
                      label: 'Configure Docker Env Variables',
                      link: '/docs/guides/docker-env/',
                    },
                    {
                      label: 'Secrets Management',
                      link: '/docs/guides/secrets/',
                    },
                    {
                      label: 'Configure Application Links',
                      link: '/docs/guides/application-links/',
                    },
                    {
                      label: 'Configuring Helm Charts for Distr Artifacts',
                      link: '/docs/guides/helm-registry-auth/',
                    },
                  ],
                },
                {
                  label: 'Operations',
                  items: [
                    {label: 'Alerts', link: '/docs/platform/alerts/'},
                    {
                      label: 'Logs & Metrics',
                      link: '/docs/platform/logs-and-metrics/',
                    },
                    {
                      label: 'Pre-flight Checks',
                      link: '/docs/guides/preflight-checks/',
                    },
                  ],
                },
              ],
            },
            {
              label: 'Container Registry',
              link: '/docs/platform/registry/',
              icon: 'seti:docker',
              items: [
                {
                  label: 'Artifact Registry',
                  link: '/docs/platform/registry/',
                },
                {
                  label: 'Set Up Your Registry',
                  link: '/docs/guides/push-to-registry/',
                },
              ],
            },
            {
              label: 'Customer Management',
              link: '/docs/platform/customer-portal/',
              icon: 'person',
              items: [
                {
                  label: 'Customer Portal',
                  link: '/docs/platform/customer-portal/',
                },
                {
                  label: 'License Management',
                  items: [
                    {
                      label: 'License Management Overview',
                      link: '/docs/platform/license-management/',
                    },
                    {
                      label: 'Application Entitlements',
                      link: '/docs/guides/application-entitlements/',
                    },
                    {
                      label: 'Artifact Entitlements',
                      link: '/docs/guides/artifact-entitlements/',
                    },
                    {
                      label: 'License Keys',
                      link: '/docs/guides/license-keys/',
                    },
                  ],
                },
                {
                  label: 'Manage Customers',
                  link: '/docs/guides/manage-customers/',
                },
                {
                  label: 'Branding & White-Labeling',
                  link: '/docs/platform/branding/',
                },
                {
                  label: 'End-Customer View of Distr',
                  link: '/docs/guides/customer-registry-access/',
                },
                {
                  label: 'Role-Based Access Control (RBAC)',
                  link: '/docs/platform/rbac/',
                },
                {
                  label: 'Subscription Management',
                  link: '/docs/platform/subscription/',
                },
                {
                  label: 'Support Bundles',
                  link: '/docs/platform/support-bundles/',
                },
              ],
            },
            {
              label: 'Automation & Developer Tools',
              link: '/docs/guides/github-actions/',
              icon: 'rocket',
              items: [
                {
                  label: 'GitHub Actions',
                  items: [
                    {
                      label: 'Automatic Deployments from GitHub',
                      link: '/docs/guides/github-actions/',
                    },
                    {
                      label: 'GitHub Action Reference',
                      link: '/docs/integrations/github-action/',
                    },
                  ],
                },
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
                {
                  label: 'Kubernetes Compatibility Matrix',
                  link: '/docs/guides/k8s-compatibility/',
                },
                {
                  label: 'Vulnerability Scanning',
                  link: '/docs/guides/vulnerability-scanning/',
                },
              ],
            },
            {
              label: 'Self-Hosting',
              link: '/docs/self-hosting/overview/',
              icon: 'laptop',
              items: [
                {label: 'Overview', link: '/docs/self-hosting/overview/'},
                {label: 'Docker Compose', link: '/docs/self-hosting/docker/'},
                {label: 'Kubernetes', link: '/docs/self-hosting/kubernetes/'},
                {
                  label: 'Feature Flags',
                  link: '/docs/self-hosting/feature-flags/',
                },
                {
                  label: 'Maintenance Jobs',
                  link: '/docs/self-hosting/maintenance/',
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
    '/docs/product/agents/': '/docs/platform/agents/',
    '/docs/product/alerts/': '/docs/platform/alerts/',
    '/docs/product/registry/': '/docs/platform/registry/',
    '/docs/product/support-bundles/': '/docs/platform/support-bundles/',
    '/docs/product/customer-portal/': '/docs/platform/customer-portal/',
    '/docs/product/branding/': '/docs/platform/branding/',
    '/docs/product/rbac/': '/docs/platform/rbac/',
    '/docs/product/license-management/': '/docs/platform/license-management/',
    '/docs/product/subscription-management/': '/docs/platform/subscription/',
    '/docs/product/distr-hub/': '/docs/platform/vendor-portal/',
    '/docs/product/faqs/': '/docs/faqs/',

    // use-cases
    '/docs/use-cases/self-managed/': '/docs/use-cases/fully-self-managed/',
    '/docs/use-cases/edge-deployments/': '/docs/use-cases/edge/',
    '/docs/use-cases/air-gapped-deployments/': '/docs/use-cases/air-gapped/',
    '/docs/use-cases/byoc-bring-your-own-cloud/': '/docs/use-cases/byoc/',

    // Guide renames
    '/docs/guides/getting-started/application/':
      '/docs/guides/create-application/',
    '/docs/guides/getting-started/deployment/':
      '/docs/guides/create-deployment/',
    '/docs/guides/getting-started/distr-on-macos/':
      '/docs/guides/setup-on-macos/',
    '/docs/guides/getting-started/how-to-registry/':
      '/docs/guides/push-to-registry/',
    '/docs/guides/configuration/docker-env/': '/docs/guides/docker-env/',
    '/docs/guides/configuration/docker-secrets/': '/docs/guides/secrets/',
    '/docs/guides/advanced/secrets/': '/docs/guides/secrets/',
    '/docs/guides/configuration/application-links/':
      '/docs/guides/application-links/',
    '/docs/guides/configuration/helm-chart-registry-auth/':
      '/docs/guides/helm-registry-auth/',
    '/docs/guides/automation/automatic-deployments-from-github/':
      '/docs/guides/github-actions/',
    '/docs/guides/automation/preflight-checks/':
      '/docs/guides/preflight-checks/',
    '/docs/guides/automation/kubernetes-compatibility-matrix/':
      '/docs/guides/k8s-compatibility/',
    '/docs/guides/automation/vulnerability-scanning/':
      '/docs/guides/vulnerability-scanning/',
    '/docs/guides/customer-management/customer-management/':
      '/docs/guides/manage-customers/',
    '/docs/guides/customer-management/end-customer-registry-view/':
      '/docs/guides/customer-registry-access/',
    '/docs/guides/customer-management/application-entitlements/':
      '/docs/guides/application-entitlements/',
    '/docs/guides/customer-management/artifact-entitlements/':
      '/docs/guides/artifact-entitlements/',
    '/docs/guides/customer-management/license-keys/':
      '/docs/guides/license-keys/',

    // Legacy guide slugs
    '/docs/guides/license-mgmt/': '/docs/guides/application-entitlements/',
    '/docs/guides/application-licenses/':
      '/docs/guides/application-entitlements/',
    '/docs/guides/artifact-licenses/': '/docs/guides/artifact-entitlements/',
    '/docs/guides/onboarding-a-new-customer/': '/docs/platform/rbac/',
    '/docs/guides/onboarding-a-docker-app/': '/docs/guides/create-application/',
    '/docs/guides/onboarding-a-helm-app/': '/docs/guides/create-application/',

    // Integration renames
    '/docs/integrations/mcp/': '/docs/',
    '/docs/integrations/api/': '/docs/integrations/rest-api/',
    '/docs/integrations/gh-action/': '/docs/integrations/github-action/',
    '/docs/integrations/personal-access-token/':
      '/docs/integrations/access-tokens/',

    // Self-hosting rename
    '/docs/self-hosting/getting-started/': '/docs/self-hosting/overview/',

    // Deleted index pages
    '/docs/integrations/': '/docs/integrations/rest-api/',
    '/docs/self-hosting/': '/docs/self-hosting/overview/',
    '/docs/guides/': '/docs/quickstart/',
  },
});
