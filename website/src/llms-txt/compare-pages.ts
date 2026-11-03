import type {AstroIntegration} from 'astro';
import {matches, select, selectAll} from 'hast-util-select';
import {appendFileSync, readFileSync} from 'node:fs';
import rehypeParse from 'rehype-parse';
import rehypeRemark from 'rehype-remark';
import remarkGfm from 'remark-gfm';
import remarkStringify from 'remark-stringify';
import {unified} from 'unified';
import {remove} from 'unist-util-remove';
import {comparePages} from './options';

type Element = NonNullable<ReturnType<typeof select>>;

const parser = unified().use(rehypeParse);
const toMarkdown = unified()
  .use(rehypeRemark)
  .use(remarkGfm)
  .use(remarkStringify);

function pageToMarkdown(html: string, withFaqs: boolean): string {
  const page = parser.parse(html);
  const content = select('article .prose', page);
  if (!content) {
    throw new Error('the compare page has no article content');
  }
  const children = [...content.children];
  const faqs = withFaqs ? selectAll('details', page) : [];
  if (faqs.length > 0) {
    children.push({
      type: 'element',
      tagName: 'h2',
      properties: {},
      children: [{type: 'text', value: 'Frequently Asked Questions'}],
    } satisfies Element);
    for (const faq of faqs) {
      children.push(
        ...[select('summary h3', faq), select('.faq-answer', faq)].filter(
          element => element !== undefined,
        ),
      );
    }
  }
  const root = {type: 'root' as const, children};
  // Testimonials and call-to-action buttons are marked not-prose, and carry no content.
  remove(root, node => matches('.not-prose, svg', node as Element));
  return String(toMarkdown.stringify(toMarkdown.runSync(root))).trim();
}

// Collapses whitespace outside code fences, the way starlight-llms-txt minifies the docs in llms-small.txt.
const minify = (markdown: string) =>
  markdown
    .split(/(```[\s\S]*?```)/)
    .map((part, index) => (index % 2 ? part : part.replace(/\s+/g, ' ')))
    .join('')
    .trim();

/**
 * Appends the compare pages to the llms-full.txt and llms-small.txt that starlight-llms-txt generates.
 * The plugin only reads the Starlight docs collection, so it lists these pages in llms.txt but cannot
 * render them. llms-small.txt leaves out the FAQs, as the plugin drops <details> there.
 */
export function llmsTxtComparePages(): AstroIntegration {
  return {
    name: 'llms-txt-compare-pages',
    hooks: {
      'astro:build:done': ({dir, logger}) => {
        const pages = comparePages().map(({id, data}) => ({
          title: data.title,
          description: data.description,
          html: readFileSync(new URL(`compare/${id}/index.html`, dir), 'utf8'),
        }));
        for (const [file, small] of [
          ['llms-full.txt', false],
          ['llms-small.txt', true],
        ] as const) {
          const segments = pages.map(({title, description, html}) => {
            const markdown = pageToMarkdown(html, !small);
            return [
              `# ${title}`,
              ...(description ? [`> ${description}`] : []),
              small ? minify(markdown) : markdown,
            ].join('\n\n');
          });
          appendFileSync(
            new URL(file, dir),
            segments.map(segment => `\n\n${segment}`).join(''),
          );
          logger.info(`Appended ${pages.length} compare pages to ${file}`);
        }
      },
    },
  };
}
