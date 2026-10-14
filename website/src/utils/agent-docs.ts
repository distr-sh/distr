import {getCollection, type CollectionEntry} from 'astro:content';
import {readFileSync} from 'node:fs';
import {dirname, extname, resolve} from 'node:path';

export const SITE = 'https://distr.sh';

type DocsEntry = CollectionEntry<'docs'>;

export async function getAgentDocs(): Promise<DocsEntry[]> {
  const entries = await getCollection(
    'docs',
    ({id, body}) =>
      (id === 'docs' || id.startsWith('docs/')) && Boolean(body?.trim()),
  );
  return entries.sort((a, b) =>
    (a.filePath ?? '').localeCompare(b.filePath ?? ''),
  );
}

export function markdownUrl(id: string): string {
  return `${SITE}/${id}.md`;
}

export function htmlUrl(id: string): string {
  return `${SITE}/${id}/`;
}

export function toAgentMarkdown(
  entry: DocsEntry,
  knownIds: Set<string>,
): string {
  const rawImports = readRawImports(entry);
  let body = entry.body ?? '';

  body = body.replace(/^import\s[^\n]*$/gm, '');
  body = body.replace(/\{\/\*[\s\S]*?\*\/\}/g, '');
  body = body.replace(
    /<Aside([^>]*)>([\s\S]*?)<\/Aside>/g,
    (_, attributes: string, content: string) =>
      blockquote(
        attribute(attributes, 'title') ??
          capitalize(attribute(attributes, 'type') ?? 'note'),
        content,
      ),
  );
  body = body.replace(
    /^:::(\w+)(?:\[([^\]]*)\])?\n([\s\S]*?)^:::$/gm,
    (_, type: string, title: string | undefined, content: string) =>
      blockquote(title ?? capitalize(type), content),
  );
  body = body.replace(
    /<TabItem([^>]*)>([\s\S]*?)<\/TabItem>/g,
    (_, attributes: string, content: string) =>
      `**${attribute(attributes, 'label') ?? ''}**\n\n${dedent(content)}\n`,
  );
  body = body.replace(/<Steps>([\s\S]*?)<\/Steps>/g, (_, content: string) =>
    dedent(content),
  );
  body = body.replace(/<\/?Tabs>/g, '');
  body = body.replace(/<Code([^>]*)\/>/g, (match, attributes: string) => {
    const variable = /code=\{(\w+)\}/.exec(attributes)?.[1];
    const code = variable ? rawImports.get(variable) : undefined;
    if (!code) {
      return '';
    }
    const language = attribute(attributes, 'lang') ?? code.language;
    return `\`\`\`${language}\n${code.content.trimEnd()}\n\`\`\``;
  });
  body = body.replace(/<(ThemedImage|Image)\b[^>]*\/>/g, '');
  body = removeRepeatedly(body, /<iframe[\s\S]*?<\/iframe>/g);
  body = body.replace(/!\[[^\]]*\]\([^)]*\)/g, '');
  body = body.replace(
    /\]\((\/docs(?:\/[^)#\s]*)?)(#[^)\s]*)?\)/g,
    (_, path: string, hash: string | undefined) => {
      const id = path.replace(/^\//, '').replace(/\/$/, '');
      const target = knownIds.has(id) ? markdownUrl(id) : `${SITE}${path}`;
      return `](${target}${hash ?? ''})`;
    },
  );
  body = body.replace(/\]\(\/(?!\/)([^)\s]*)\)/g, `](${SITE}/$1)`);
  body = body.replace(/\n{3,}/g, '\n\n').trim();

  const description = entry.data.description
    ? `> ${entry.data.description}\n\n`
    : '';
  return `# ${entry.data.title}\n\n${description}Source: ${htmlUrl(entry.id)}\n\n${body}\n`;
}

function readRawImports(
  entry: DocsEntry,
): Map<string, {content: string; language: string}> {
  const imports = new Map<string, {content: string; language: string}>();
  if (!entry.filePath) {
    return imports;
  }
  for (const [, variable, path] of (entry.body ?? '').matchAll(
    /^import\s+(\w+)\s+from\s+'([^']+)\?raw';?$/gm,
  )) {
    const file = resolve(dirname(entry.filePath), path);
    imports.set(variable, {
      content: readFileSync(file, 'utf8'),
      language: extname(file).slice(1) || 'text',
    });
  }
  return imports;
}

function attribute(attributes: string, name: string): string | undefined {
  return new RegExp(`${name}="([^"]*)"`).exec(attributes)?.[1];
}

function blockquote(title: string, content: string): string {
  const lines = dedent(content).split('\n');
  return [
    `> **${title}:**`,
    ...lines.map(line => (line ? `> ${line}` : '>')),
  ].join('\n');
}

function dedent(content: string): string {
  const lines = content.replace(/^\n+|\s+$/g, '').split('\n');
  const indent = Math.min(
    ...lines
      .filter(line => line.trim())
      .map(line => /^ */.exec(line)![0].length),
  );
  return lines
    .map(line => line.slice(Number.isFinite(indent) ? indent : 0))
    .join('\n');
}

// A single pass can join the remains of two matches into a new one.
function removeRepeatedly(value: string, pattern: RegExp): string {
  let previous: string;
  do {
    previous = value;
    value = value.replace(pattern, '');
  } while (value !== previous);
  return value;
}

function capitalize(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
