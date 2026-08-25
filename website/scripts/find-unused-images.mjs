#!/usr/bin/env node
import fs from 'node:fs/promises';
import path from 'node:path';
import process from 'node:process';
import {fileURLToPath} from 'node:url';

const websiteRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '..',
);
// Only 'assetsDir' is ever pruned. The search roots are read to collect
// references, so 'public' is never a deletion candidate: it is served verbatim
// by URL and its files can be requested from outside the repository.
const assetsDir = path.join(websiteRoot, 'src', 'assets');
const searchRoots = ['src', 'scripts', 'public', 'astro.config.mjs'];
const ignoredDirectories = new Set(['node_modules', 'dist', '.astro', '.git']);
const imageExtensions = ['png', 'jpg', 'jpeg', 'webp', 'svg', 'gif', 'avif'];
const sourceExtensions = new Set([
  '.astro',
  '.cjs',
  '.css',
  '.html',
  '.js',
  '.json',
  '.md',
  '.mdx',
  '.mjs',
  '.scss',
  '.ts',
  '.tsx',
  '.txt',
  '.xml',
  '.yaml',
  '.yml',
]);

const imageReference = new RegExp(
  String.raw`(?:~/|\.{1,2}/|/|\bsrc/)[\w.~@/-]*?\.(?:${imageExtensions.join('|')})`,
  'gi',
);

async function walk(target) {
  const stats = await fs.stat(target).catch(() => undefined);
  if (!stats) {
    return [];
  }
  if (stats.isFile()) {
    return [target];
  }
  const entries = await fs.readdir(target, {withFileTypes: true});
  const files = [];
  for (const entry of entries) {
    if (entry.isDirectory() && ignoredDirectories.has(entry.name)) {
      continue;
    }
    files.push(...(await walk(path.join(target, entry.name))));
  }
  return files;
}

function resolveReference(reference, sourceFile) {
  if (reference.startsWith('~/')) {
    return path.join(websiteRoot, 'src', reference.slice(2));
  }
  // Content frontmatter addresses images as '/src/assets/...', which Astro
  // resolves from the project root rather than from 'public'.
  if (reference.startsWith('/')) {
    return path.join(websiteRoot, reference.slice(1));
  }
  if (reference.startsWith('src/')) {
    return path.join(websiteRoot, reference);
  }
  return path.resolve(path.dirname(sourceFile), reference);
}

async function removeEmptyParents(file) {
  let directory = path.dirname(file);
  while (directory.startsWith(assetsDir + path.sep)) {
    const remaining = await fs.readdir(directory);
    if (remaining.length > 0) {
      return;
    }
    await fs.rmdir(directory);
    directory = path.dirname(directory);
  }
}

const shouldDelete = process.argv.slice(2).includes('--delete');

const images = (await walk(assetsDir)).filter(file =>
  imageExtensions.includes(path.extname(file).slice(1).toLowerCase()),
);
const sourceFiles = (
  await Promise.all(searchRoots.map(root => walk(path.join(websiteRoot, root))))
).flat();

const referenced = new Set();
for (const file of sourceFiles) {
  if (!sourceExtensions.has(path.extname(file).toLowerCase())) {
    continue;
  }
  const content = await fs.readFile(file, 'utf8');
  for (const [reference] of content.matchAll(imageReference)) {
    referenced.add(resolveReference(reference, file));
  }
}

const unused = images
  .filter(image => !referenced.has(image))
  .map(image => path.relative(websiteRoot, image))
  .sort();

if (unused.length === 0) {
  console.log(`All ${images.length} images in src/assets are referenced.`);
  process.exit(0);
}

for (const image of unused) {
  console.log(`${shouldDelete ? 'deleting' : 'unreferenced'} ${image}`);
}

if (!shouldDelete) {
  console.error(
    `${unused.length} unreferenced image(s) found. Run "pnpm images:prune" to delete them.`,
  );
  process.exit(1);
}

for (const image of unused) {
  const file = path.join(websiteRoot, image);
  await fs.unlink(file);
  await removeEmptyParents(file);
}
console.log(`Deleted ${unused.length} unreferenced image(s).`);
