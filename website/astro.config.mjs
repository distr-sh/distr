// @ts-check
import {unified} from '@astrojs/markdown-remark';
import mdx from '@astrojs/mdx';
import preact from '@astrojs/preact';
import sitemap from '@astrojs/sitemap';
import starlight from '@astrojs/starlight';
import tailwindcss from '@tailwindcss/vite';
import icon from 'astro-icon';
import {defineConfig, fontProviders} from 'astro/config';
import rehypeExternalLinks from 'rehype-external-links';
import rehypeMermaid from 'rehype-mermaid';
import starlightLinksValidator from 'starlight-links-validator';
import starlightSidebarTopics from 'starlight-sidebar-topics';

// https://astro.build/config
export default defineConfig({
  site: 'https://distr.sh',
  prefetch: {
    prefetchAll: true,
  },
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
          icon: 'comment-alt',
          label: 'GitHub Discussions',
          href: 'https://github.com/distr-sh/distr/discussions',
        },
      ],
      components: {
        // Components can be overwritten here
        Head: './src/components/overwrites/Head.astro',
        Header: './src/components/overwrites/Header.astro',
        PageTitle: './src/components/overwrites/PageTitle.astro',
        ContentPanel: './src/components/overwrites/ContentPanel.astro',
        Footer: './src/components/overwrites/Footer.astro',
        Sidebar: './src/components/overwrites/Sidebar.astro',
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
                      autogenerate: {
                        directory:
                          'docs/getting-started/distribution-scenarios',
                      },
                    },
                  ],
                },
                {
                  label: 'Account',
                  items: [
                    {autogenerate: {directory: 'docs/getting-started/account'}},
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
                  items: [{autogenerate: {directory: 'docs/agents/overview'}}],
                },
                {
                  label: 'Configuration',
                  items: [
                    {autogenerate: {directory: 'docs/agents/configuration'}},
                  ],
                },
                {
                  label: 'Monitoring',
                  items: [
                    {autogenerate: {directory: 'docs/agents/monitoring'}},
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
                    {autogenerate: {directory: 'docs/registry/overview'}},
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
                    {autogenerate: {directory: 'docs/platform/licenses'}},
                  ],
                },
                {
                  label: 'Support',
                  items: [{autogenerate: {directory: 'docs/platform/support'}}],
                },
                {
                  label: 'Customer Portal',
                  items: [
                    {
                      autogenerate: {
                        directory: 'docs/platform/customer-portal',
                      },
                    },
                  ],
                },
                {
                  label: 'Customer Management',
                  items: [
                    {autogenerate: {directory: 'docs/platform/customers'}},
                  ],
                },
                {
                  label: 'User Management',
                  items: [
                    {
                      autogenerate: {
                        directory: 'docs/platform/user-management',
                      },
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
                  label: 'Air-Gapped Packaging',
                  items: [
                    {
                      label: 'Air-Gapped Deployments with Zarf',
                      link: '/docs/integrations/zarf/',
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
                  label: 'Overview',
                  items: [{autogenerate: {directory: 'docs/self-hosting'}}],
                },
              ],
            },
          ],
          {
            exclude: ['**/privacy-policy', '**/404', '**/changelog'],
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
    processor: unified({
      rehypePlugins: [
        [rehypeMermaid, {strategy: 'inline-svg'}],
        [
          rehypeExternalLinks,
          {target: '_blank', rel: ['noopener', 'noreferrer']},
        ],
      ],
    }),
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

    // Self-hosting redirects
    '/docs/self-hosting/getting-started/': '/docs/self-hosting/',

    // Legacy blog redirects (content superseded by /compare/ pages)
    '/blog/distr-vs-replicated/': '/compare/replicated/',

    // Renamed blog posts
    '/blog/self-managed-vs-cloud-vs-byoc/':
      '/blog/self-hosted-vs-saas-vs-byoc/',

    // Glossary redirects
    '/glossary/self-managed-software/': '/glossary/self-hosted-software/',
  },
});
