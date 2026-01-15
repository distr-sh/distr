// @ts-check
import mdx from '@astrojs/mdx';
import preact from '@astrojs/preact';
import sitemap from '@astrojs/sitemap';
import starlight from '@astrojs/starlight';
import starlightUtils from '@lorenzo_lewis/starlight-utils';
import tailwindcss from '@tailwindcss/vite';
import icon from 'astro-icon';
import {defineConfig} from 'astro/config';
import serviceWorker from 'astrojs-service-worker';
import rehypeMermaid from 'rehype-mermaid';
import starlightLinksValidator from 'starlight-links-validator';

// https://astro.build/config
export default defineConfig({
  site: 'https://distr.sh',

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
        baseUrl: 'https://github.com/glasskube/distr.sh/tree/main',
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
          href: 'https://github.com/glasskube/distr',
        },
        {
          icon: 'discord',
          label: 'Discord',
          href: 'https://discord.gg/6qqBSAWZfW',
        },
      ],
      components: {
        // Components can be overwritten here
        PageTitle: './src/components/overwrites/PageTitle.astro',
        ContentPanel: './src/components/overwrites/ContentPanel.astro',
        Footer: './src/components/overwrites/Footer.astro',
        SocialIcons: './src/components/overwrites/SocialIcons.astro',
      },
      sidebar: [
        {
          label: 'Navbar',
          items: [
            {label: 'Docs', link: '/docs/getting-started/what-is-distr/'},
            {label: 'Pricing', link: '/pricing/'},
            {label: 'Blog', link: '/blog/'},
            {label: 'Case Studies', link: '/case-studies/'},
          ],
        },
        {
          label: 'Getting started',
          autogenerate: {directory: 'docs/getting-started'},
        },
        {
          label: 'Use cases',
          autogenerate: {directory: 'docs/use-cases'},
        },
        {
          label: 'Guides',
          autogenerate: {directory: 'docs/guides'},
        },
        {
          label: 'Product',
          autogenerate: {directory: 'docs/product'},
        },
        {
          label: 'Self hosting',
          autogenerate: {directory: 'docs/self-hosting'},
        },
        {
          label: 'Integrations',
          autogenerate: {directory: 'docs/integrations'},
        },
        {
          label: 'API Reference',
          link: 'https://app.distr.sh/docs',
        },
        {
          label: 'FAQs',
          link: '/docs/faqs/',
        },
      ],
      tableOfContents: {
        minHeadingLevel: 2,
        maxHeadingLevel: 6,
      },
      prerender: true,
      plugins: [
        starlightLinksValidator({
          exclude: [
            '/',
            '/contact/',
            '/pricing/',
            '/blog/**',
            '/glossary/**',
            '/get-started/',
            '/onboarding/',
          ],
        }),
        starlightUtils({
          navLinks: {
            leading: {useSidebarLabelled: 'Navbar'},
          },
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
    '/docs/getting-started/': '/docs/',
    '/docs/getting-started/about/': '/docs/',
    '/docs/getting-started/what-is-distr/': '/docs/',
    '/docs/getting-started/how-it-works/': '/docs/core-concepts/',
    '/docs/getting-started/core-concepts/': '/docs/core-concepts/',
    '/docs/getting-started/quickstart/': '/docs/quickstart/',
    '/docs/getting-started/deployment-methods/': '/docs/subscription/',
    '/docs/product/distr-hub/': '/docs/product/vendor-portal/',
    '/docs/use-cases/self-managed/': '/docs/use-cases/fully-self-managed/',
    '/docs/use-cases/byoc/': '/docs/use-cases/byoc-bring-your-own-cloud/',
    '/docs/product/faqs/': '/docs/faqs/',
    '/docs/privacy-policy/': '/privacy-policy/',
    '/docs/guides/license-mgmt/': '/docs/guides/application-licenses/',
    '/docs/guides/onboarding-a-new-customer/': '/docs/product/rbac/',
    '/docs/guides/onboarding-a-docker-app/': '/docs/guides/application/',
    '/docs/guides/onboarding-a-helm-app/': '/docs/guides/application/',
  },
});
