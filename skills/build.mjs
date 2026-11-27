#!/usr/bin/env node

// Builds the skill for a release into dist/skill:
//   assets/  the archives uploaded to the GitHub release of the tag
//   site/    the files synced to distr.sh, whose index and marketplace point at those archives

import {zipSync} from 'fflate';
import {createHash} from 'node:crypto';
import {mkdirSync, readdirSync, readFileSync, rmSync, statSync, writeFileSync} from 'node:fs';
import {dirname, join} from 'node:path';
import {parse} from 'yaml';

const skillDir = join(import.meta.dirname, 'distr');
const templatesDir = join(import.meta.dirname, 'templates');
const outDir = join(import.meta.dirname, '..', 'dist', 'skill');
const assetsDir = join(outDir, 'assets');
const siteDir = join(outDir, 'site', '.well-known', 'agent-skills');

function readFrontmatter() {
  const content = readFileSync(join(skillDir, 'SKILL.md'), 'utf8');
  const match = content.match(/^---\n([\s\S]*?)\n---\n/);
  if (!match) throw new Error('SKILL.md has no frontmatter');
  return parse(match[1]);
}

function listSkillFiles() {
  return readdirSync(skillDir, {recursive: true})
    .filter((path) => statSync(join(skillDir, path)).isFile())
    .sort();
}

const tag = process.argv[2] ?? process.env.VERSION;
if (!tag) {
  console.error('usage: build.mjs <tag>');
  process.exit(1);
}

const releaseUrl = `https://github.com/distr-sh/distr/releases/download/${tag}`;
const {name, description} = readFrontmatter();

// A fixed mtime keeps the archives from depending on when the files were checked out.
const mtime = new Date('2020-01-01T00:00:00Z');

function zip(files) {
  return zipSync(Object.fromEntries(Object.entries(files).map(([path, content]) => [path, [content, {mtime}]])));
}

function sha256(bytes) {
  return createHash('sha256').update(bytes).digest('hex');
}

// Replaces every "{{key}}" string in a JSON template with the JSON encoding of values[key].
function render(template, values) {
  return readFileSync(join(templatesDir, template), 'utf8').replace(/"\{\{(\w+)\}\}"/g, (_, key) => {
    if (!(key in values)) throw new Error(`${template}: no value for {{${key}}}`);
    return JSON.stringify(values[key]);
  });
}

function writeFiles(dir, files) {
  for (const [path, content] of Object.entries(files)) {
    mkdirSync(dirname(join(dir, path)), {recursive: true});
    writeFileSync(join(dir, path), content);
  }
}

const skillFiles = Object.fromEntries(listSkillFiles().map((path) => [path, readFileSync(join(skillDir, path))]));

// A release candidate is tagged before release-please bumps metadata.version, so the shipped
// SKILL.md takes its version from the tag.
const versionLine = /^(\s+version: )'[^']*'/m;
if (!versionLine.test(skillFiles['SKILL.md'])) throw new Error('SKILL.md has no metadata.version');
skillFiles['SKILL.md'] = Buffer.from(skillFiles['SKILL.md'].toString().replace(versionLine, `$1'${tag}'`));

const pluginFiles = {
  '.claude-plugin/plugin.json': Buffer.from(render('plugin.json', {})),
  ...Object.fromEntries(Object.entries(skillFiles).map(([path, content]) => [`skills/${name}/${path}`, content])),
};

rmSync(outDir, {recursive: true, force: true});

const archive = zip(skillFiles);
const plugin = zip(pluginFiles);
writeFiles(assetsDir, {[`${name}.zip`]: archive, [`${name}-claude-plugin.zip`]: plugin});
writeFiles(siteDir, {
  'index.json': render('index.json', {
    name,
    description,
    archiveUrl: `${releaseUrl}/${name}.zip`,
    archiveDigest: `sha256:${sha256(archive)}`,
  }),
  'marketplace.json': render('marketplace.json', {
    name,
    description,
    version: tag,
    pluginUrl: `${releaseUrl}/${name}-claude-plugin.zip`,
    pluginSha256: sha256(plugin),
  }),
  [`${name}/SKILL.md`]: skillFiles['SKILL.md'],
});
