#!/usr/bin/env node
// Usage: node scripts/changed-pages.mjs <deployed-dir> > changed-pages.txt
//
// Prints every URL of dist/sitemap-*.xml whose page differs from its copy in
// <deployed-dir>, which the deploy fills with the HTML and sitemaps that are
// live before the sync. It also writes <lastmod> into the sitemap in dist: the
// current time for a changed page, the deployed value for an unchanged one. The
// live sitemap is therefore the only state, and a page only gets a date once it
// is seen to change, since Bing ignores dates that all sit on the same day.
import {parse} from 'node-html-parser';
import fs from 'node:fs/promises';
import path from 'node:path';
import process from 'node:process';
import {fileURLToPath} from 'node:url';

const distDir = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '..',
  'dist',
);

const sitemapEntry = /<loc>([^<]*)<\/loc>(?:<lastmod>([^<]*)<\/lastmod>)?/g;

// Astro fingerprints emitted assets, so a dependency bump changes the markup of
// every page without changing its content.
const assetReference = /_astro\/[^"'\s),]+/g;

async function readSitemaps(dir) {
  const names = (await fs.readdir(dir).catch(() => [])).filter(name =>
    /^sitemap-\d+\.xml$/.test(name),
  );
  const sitemaps = new Map();
  for (const name of names) {
    sitemaps.set(name, await fs.readFile(path.join(dir, name), 'utf8'));
  }
  return sitemaps;
}

async function readPage(dir, url) {
  const relative = new URL(url).pathname.replace(/^\/+|\/+$/g, '');
  const file = relative.endsWith('.html')
    ? path.join(dir, relative)
    : path.join(dir, relative, 'index.html');
  return fs.readFile(file, 'utf8').catch(() => undefined);
}

function normalize(html) {
  const document = parse(html);
  // The JSON-LD scripts stay, their dateModified is content. The generator meta
  // tags carry the Astro and Starlight versions.
  for (const element of document.querySelectorAll(
    'style, script:not([type="application/ld+json"]), meta[name="generator"]',
  )) {
    element.remove();
  }
  // Astro derives an island's uid from the build, not from the page.
  for (const element of document.querySelectorAll('astro-island')) {
    element.removeAttribute('uid');
  }
  return document.toString().replaceAll(assetReference, '_astro/');
}

const deployedDir = process.argv[2];
if (!deployedDir) {
  console.error('Usage: node scripts/changed-pages.mjs <deployed-dir>');
  process.exit(1);
}

const deployedLastmod = new Map();
for (const xml of (await readSitemaps(deployedDir)).values()) {
  for (const [, url, lastmod] of xml.matchAll(sitemapEntry)) {
    if (lastmod) {
      deployedLastmod.set(url, lastmod);
    }
  }
}

const now = new Date().toISOString().replace(/\.\d+Z$/, 'Z');
const sitemaps = await readSitemaps(distDir);
const lastmodByUrl = new Map();
const changed = [];
let pageCount = 0;
for (const xml of sitemaps.values()) {
  for (const [, url] of xml.matchAll(sitemapEntry)) {
    pageCount++;
    const page = await readPage(distDir, url);
    if (page === undefined) {
      throw new Error(`The sitemap lists ${url}, which has no built page.`);
    }
    const deployed = await readPage(deployedDir, url);
    if (deployed === undefined || normalize(deployed) !== normalize(page)) {
      changed.push(url);
      lastmodByUrl.set(url, now);
    } else if (deployedLastmod.has(url)) {
      lastmodByUrl.set(url, deployedLastmod.get(url));
    }
  }
}

for (const [name, xml] of sitemaps) {
  await fs.writeFile(
    path.join(distDir, name),
    xml.replaceAll(sitemapEntry, (_match, url) =>
      lastmodByUrl.has(url)
        ? `<loc>${url}</loc><lastmod>${lastmodByUrl.get(url)}</lastmod>`
        : `<loc>${url}</loc>`,
    ),
  );
}

console.error(`${changed.length} of ${pageCount} pages changed.`);
for (const url of changed) {
  console.log(url);
}
