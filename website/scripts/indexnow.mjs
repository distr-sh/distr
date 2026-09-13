#!/usr/bin/env node
import {parse} from 'node-html-parser';
import {execFile} from 'node:child_process';
import {createHash} from 'node:crypto';
import fs from 'node:fs/promises';
import path from 'node:path';
import process from 'node:process';
import {fileURLToPath} from 'node:url';
import {promisify} from 'node:util';

const run = promisify(execFile);

const websiteRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '..',
);
const distDir = path.join(websiteRoot, 'dist');
const stateDir = path.join(websiteRoot, '.indexnow');
const pendingFile = path.join(stateDir, 'pending.json');
const manifestFile = path.join(stateDir, 'manifest.json');
const manifestUri = 's3://distr.sh/.indexnow/manifest.json';
const submissionEndpoint = 'https://api.indexnow.org/indexnow';
const host = 'distr.sh';
const manifestVersion = 1;
// IndexNow rejects requests carrying more than 10000 URLs.
const maxUrlsPerRequest = 10000;

const sitemapFilePattern = /^sitemap-\d+\.xml$/;
const sitemapIndexFile = 'sitemap-index.xml';
const locationPattern = /<loc>([^<]*)<\/loc>/g;
// Consuming a <lastmod> that is already there keeps a rerun over the same
// dist from stacking a second one onto every entry.
const locationEntryPattern =
  /<loc>([^<]*)<\/loc>(?:<lastmod>[^<]*<\/lastmod>)?/g;

// Astro fingerprints emitted assets, so a dependency bump changes every page's
// markup without changing a word of its content. Both forms are collapsed
// before hashing: '_astro/print.ehPL0gv-.css' keeps its name and extension,
// while a font file is named after its hash alone and keeps nothing.
const hashedFontReference = /_astro\/fonts\/[A-Za-z0-9_-]+(\.[A-Za-z0-9]+)/g;
const hashedAssetReference =
  /(_astro\/[A-Za-z0-9._/-]*?)\.[A-Za-z0-9_-]{8,}(\.[A-Za-z0-9]+)(?![A-Za-z0-9._-])/g;

function printHelp() {
  console.log(`Usage: node scripts/indexnow.mjs <prepare|submit> [--dry-run]

prepare  Hash every page the sitemap lists, diff it against the manifest of the
         previous deploy, write the resulting lastmod values into the sitemap
         and record the changed URLs for "submit". Run after "pnpm build" and
         before the S3 sync.
submit   Send the recorded URLs to IndexNow and persist the new manifest. Run
         after the S3 sync and the CloudFront invalidation.

--dry-run  Keep everything local: read and write the manifest in
           website/.indexnow instead of S3, and submit nothing.

See scripts/indexnow.md.`);
}

function decodeXmlText(value) {
  return value
    .replaceAll('&lt;', '<')
    .replaceAll('&gt;', '>')
    .replaceAll('&quot;', '"')
    .replaceAll('&apos;', "'")
    .replaceAll('&amp;', '&');
}

function distFileForUrl(url) {
  const relative = new URL(url).pathname.replace(/^\/+|\/+$/g, '');
  return relative.endsWith('.html')
    ? path.join(distDir, relative)
    : path.join(distDir, relative, 'index.html');
}

function contentHash(html) {
  const document = parse(html);
  for (const element of document.querySelectorAll('style')) {
    element.remove();
  }
  for (const element of document.querySelectorAll('script')) {
    // Glossary and CRA pages carry a JSON-LD 'dateModified', which is content.
    if (element.getAttribute('type') !== 'application/ld+json') {
      element.remove();
    }
  }
  const normalized = document
    .toString()
    .replaceAll(hashedFontReference, '_astro/fonts/font$1')
    .replaceAll(hashedAssetReference, '$1$2');
  return createHash('sha256').update(normalized).digest('hex');
}

// The sitemap schema fixes the order of a <url>'s children, so <lastmod> has to
// follow <loc> rather than being appended.
function withLastmod(xml, lastmodFor) {
  return xml.replaceAll(locationEntryPattern, (_match, location) => {
    const lastmod = lastmodFor(decodeXmlText(location));
    const entry = `<loc>${location}</loc>`;
    return lastmod ? `${entry}<lastmod>${lastmod}</lastmod>` : entry;
  });
}

async function readSitemap() {
  const names = (await fs.readdir(distDir).catch(() => [])).filter(name =>
    sitemapFilePattern.test(name),
  );
  if (names.length === 0) {
    throw new Error(`No sitemap in ${distDir}. Run "pnpm build" first.`);
  }
  const urls = new Set();
  for (const name of names.sort()) {
    const xml = await fs.readFile(path.join(distDir, name), 'utf8');
    for (const [, location] of xml.matchAll(locationPattern)) {
      urls.add(decodeXmlText(location));
    }
  }
  return {names, urls: [...urls]};
}

async function hashPages(urls) {
  const hashes = new Map();
  const missing = [];
  for (const url of urls) {
    const file = distFileForUrl(url);
    const html = await fs.readFile(file, 'utf8').catch(() => undefined);
    if (html === undefined) {
      missing.push(`${url} (expected ${path.relative(websiteRoot, file)})`);
      continue;
    }
    hashes.set(url, contentHash(html));
  }
  if (missing.length > 0) {
    throw new Error(
      `The sitemap lists ${missing.length} URL(s) without a built page:\n  ${missing.join('\n  ')}`,
    );
  }
  return hashes;
}

function parseManifest(contents, source) {
  const manifest = JSON.parse(contents);
  if (manifest.version !== manifestVersion) {
    throw new Error(
      `Manifest at ${source} has version ${manifest.version}, expected ${manifestVersion}.`,
    );
  }
  return manifest;
}

async function readManifest(dryRun) {
  if (dryRun) {
    const contents = await fs.readFile(manifestFile, 'utf8').catch(() => '');
    if (contents === '') {
      console.log(
        `No manifest at ${path.relative(websiteRoot, manifestFile)}.`,
      );
      return undefined;
    }
    return parseManifest(contents, manifestFile);
  }
  let stdout;
  try {
    ({stdout} = await run('aws', ['s3', 'cp', manifestUri, '-']));
  } catch (error) {
    // S3 answers a missing key with 403 rather than 404 when the caller has no
    // ListBucket permission, so a failed read cannot be told apart from a first
    // run. Reading it as a first run is the safe choice because it submits
    // nothing; a real permission problem surfaces when "submit" writes back.
    console.log(
      `No manifest read from ${manifestUri}, treating this as a first run.`,
    );
    console.log(`  ${String(error.stderr ?? error.message).trim()}`);
    return undefined;
  }
  return parseManifest(stdout, manifestUri);
}

async function writeManifest(manifest, dryRun) {
  await fs.mkdir(stateDir, {recursive: true});
  await fs.writeFile(manifestFile, `${JSON.stringify(manifest, null, 2)}\n`);
  const count = Object.keys(manifest.urls).length;
  if (dryRun) {
    console.log(
      `Wrote ${count} entries to ${path.relative(websiteRoot, manifestFile)}.`,
    );
    return;
  }
  // No --acl, so the manifest stays private: the deploy grants public-read to
  // the site's own objects only.
  await run('aws', [
    's3',
    'cp',
    manifestFile,
    manifestUri,
    '--content-type',
    'application/json',
  ]);
  console.log(`Wrote ${count} entries to ${manifestUri}.`);
}

async function prepare(dryRun) {
  const now = new Date().toISOString().replace(/\.\d+Z$/, 'Z');
  const sitemap = await readSitemap();
  const hashes = await hashPages(sitemap.urls);
  const previous = await readManifest(dryRun);
  const isFirstRun = previous === undefined;
  const previousUrls = previous?.urls ?? {};

  const changed = [];
  const urls = {};
  for (const [url, hash] of hashes) {
    const before = previousUrls[url];
    if (before?.hash === hash) {
      urls[url] = {hash, lastmod: before.lastmod};
      continue;
    }
    // A first run has nothing to compare against, and IndexNow asks for changes
    // made after adoption only. Leaving lastmod unset until a change is really
    // observed also keeps Bing from seeing every page dated the same day, which
    // it treats as a reason to disregard the dates.
    urls[url] = {hash, lastmod: isFirstRun ? null : now};
    if (!isFirstRun) {
      changed.push(url);
    }
  }
  // Engines drop a page once they see it gone, so removals are worth sending.
  const removed = Object.keys(previousUrls).filter(url => !hashes.has(url));

  const lastmodByUrl = new Map(
    Object.entries(urls)
      .filter(([, entry]) => entry.lastmod !== null)
      .map(([url, entry]) => [url, entry.lastmod]),
  );
  for (const name of sitemap.names) {
    const file = path.join(distDir, name);
    const xml = await fs.readFile(file, 'utf8');
    await fs.writeFile(
      file,
      withLastmod(xml, url => lastmodByUrl.get(url)),
    );
  }
  const newest = [...lastmodByUrl.values()].sort().at(-1);
  const indexFile = path.join(distDir, sitemapIndexFile);
  const indexXml = await fs.readFile(indexFile, 'utf8').catch(() => undefined);
  if (indexXml !== undefined) {
    await fs.writeFile(
      indexFile,
      withLastmod(indexXml, () => newest),
    );
  }

  const manifest = {version: manifestVersion, urls};
  await fs.mkdir(stateDir, {recursive: true});
  await fs.writeFile(
    pendingFile,
    `${JSON.stringify({urls: [...changed, ...removed], manifest}, null, 2)}\n`,
  );

  console.log(
    `${hashes.size} indexable pages, ${changed.length} changed, ${removed.length} removed, ${lastmodByUrl.size} with lastmod.`,
  );
  if (isFirstRun) {
    console.log('First run: recording the manifest without submitting.');
  }
  if (dryRun) {
    await writeManifest(manifest, dryRun);
  }
}

async function submit(dryRun) {
  const contents = await fs.readFile(pendingFile, 'utf8').catch(() => '');
  if (contents === '') {
    throw new Error(
      `No ${path.relative(websiteRoot, pendingFile)}. Run "prepare" first.`,
    );
  }
  const pending = JSON.parse(contents);
  const key = process.env.INDEXNOW_KEY;

  if (pending.urls.length === 0) {
    console.log('Nothing to submit.');
  } else if (!key) {
    // The manifest is still recorded, so the sitemap keeps getting lastmod
    // values even before the key exists.
    console.log(
      `INDEXNOW_KEY is not set, not submitting ${pending.urls.length} URL(s).`,
    );
  } else if (dryRun) {
    console.log(`Dry run, would submit:\n  ${pending.urls.join('\n  ')}`);
  } else {
    for (let i = 0; i < pending.urls.length; i += maxUrlsPerRequest) {
      const urlList = pending.urls.slice(i, i + maxUrlsPerRequest);
      const response = await fetch(submissionEndpoint, {
        method: 'POST',
        headers: {'content-type': 'application/json; charset=utf-8'},
        body: JSON.stringify({
          host,
          key,
          keyLocation: `https://${host}/${key}.txt`,
          urlList,
        }),
      });
      // 202 means the key file has not been fetched yet, which is what the
      // first submission from a new key returns.
      if (response.status !== 200 && response.status !== 202) {
        const body = (await response.text()).trim();
        console.error(`IndexNow returned ${response.status}: ${body}`);
        console.error(
          'Keeping the previous manifest so the next deploy retries these URLs.',
        );
        return;
      }
      console.log(
        `Submitted ${urlList.length} URL(s), IndexNow returned ${response.status}.`,
      );
    }
  }

  await writeManifest(pending.manifest, dryRun);
}

const args = process.argv.slice(2);
const command = args.find(argument => !argument.startsWith('-'));
const dryRun = args.includes('--dry-run');
const wantsHelp = args.includes('--help') || args.includes('-h');

if (wantsHelp || command === undefined) {
  printHelp();
  process.exit(wantsHelp ? 0 : 1);
}

try {
  if (command === 'prepare') {
    await prepare(dryRun);
  } else if (command === 'submit') {
    await submit(dryRun);
  } else {
    console.error(`Unknown command "${command}".`);
    printHelp();
    process.exit(1);
  }
} catch (error) {
  console.error(error.message ?? error);
  process.exit(1);
}
