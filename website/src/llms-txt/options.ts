import {parseFrontmatter} from '@astrojs/markdown-remark';
import {globSync, readFileSync} from 'node:fs';
import {fileURLToPath} from 'node:url';
import type starlightLlmsTxt from 'starlight-llms-txt';

type LlmsTxtOptions = NonNullable<Parameters<typeof starlightLlmsTxt>[0]>;

interface Frontmatter {
  title: string;
  description?: string;
  slug?: string;
  draft?: boolean;
}

const promote = ['docs', 'docs/core-concepts', 'docs/quickstart'];

const collator = new Intl.Collator('en');

const readText = (file: string) =>
  readFileSync(new URL(file, import.meta.url), 'utf8').trim();

// The Astro config cannot query content collections, so this reads the frontmatter itself and derives
// the entry ID the way Astro's glob loader does: the slug, else the path without extension and /index.
function readContentPages(collectionDir: string, pattern: string) {
  const dir = new URL(`../content/${collectionDir}/`, import.meta.url);
  return globSync(pattern, {cwd: fileURLToPath(dir)})
    .map(file => {
      const data = parseFrontmatter(readFileSync(new URL(file, dir), 'utf8'))
        .frontmatter as Frontmatter;
      const id =
        data.slug ?? file.replace(/\.mdx?$/, '').replace(/\/index$/, '');
      return {id, data};
    })
    .filter(page => !page.data.draft);
}

const rank = (id: string) =>
  promote.includes(id) ? promote.indexOf(id) : Infinity;

export function llmsTxtOptions(site: string): LlmsTxtOptions {
  const docsPageLinks = readContentPages('docs', 'docs/**/*.{md,mdx}')
    .sort((a, b) => rank(a.id) - rank(b.id) || collator.compare(a.id, b.id))
    .map(({id, data}) => ({
      label: data.title,
      url: `${site}/${id}/`,
      description: data.description,
    }));

  const comparePageLinks = readContentPages('compare', '*.{md,mdx}')
    .sort((a, b) => collator.compare(a.id, b.id))
    .map(({id, data}) => ({
      label: data.title,
      url: `${site}/compare/${id}/`,
      description: data.description,
    }));

  return {
    // The plugin prefixes the description with "> " once, so it has to stay on a single line.
    description: readText('description.md'),
    details: readText('details.md'),
    optionalLinks: [
      {
        label: 'API reference',
        url: 'https://app.distr.sh/docs',
        description:
          'the interactive reference of the Distr REST API on Distr Cloud',
      },
      {
        label: 'OpenAPI specification',
        url: 'https://app.distr.sh/docs/openapi.json',
        description:
          'the machine-readable specification of the Distr REST API on Distr Cloud',
      },
      ...comparePageLinks,
      ...docsPageLinks,
    ],
    customSelectors: {all: ['.sl-anchor-link']},
    promote,
    demote: [
      'changelog',
      'privacy-policy',
      'terms-of-service',
      'data-processing-agreement',
    ],
    exclude: [
      'changelog',
      'privacy-policy',
      'terms-of-service',
      'data-processing-agreement',
    ],
  };
}
