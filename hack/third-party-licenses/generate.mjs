#!/usr/bin/env node

import {execFileSync} from 'node:child_process';
import {globSync, mkdtempSync, readdirSync, readFileSync, rmSync, writeFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {dirname, join} from 'node:path';

const DIR = 'hack/third-party-licenses';
const OWN_MODULE = 'github.com/distr-sh/distr';
const OWN_IMAGES = /^(ghcr\.io\/distr-sh\/|registry\.distr\.sh\/)/;

const errors = [];
const notices = new Map();

// Apache-2.0 requires redistributions to carry the attribution notices of a dependency's NOTICE file.
function collectNotices(name, dir) {
  for (const file of readdirSync(dir).filter((file) => /^NOTICE(\.txt|\.md)?$/i.test(file))) {
    const path = join(dir, file);
    if (!notices.has(path)) {
      const text = readFileSync(path, 'utf8')
        .replace(/\r\n/g, '\n')
        .replace(/[ \t]+$/gm, '')
        .replace(/\n{3,}/g, '\n\n')
        .trim();
      notices.set(path, {name, text});
    }
  }
}

function readJSON(file) {
  return JSON.parse(readFileSync(join(DIR, file), 'utf8'));
}

function link(text, url) {
  return url ? `[${text}](${url})` : text;
}

function table(header, rows) {
  return [
    `| ${header.join(' | ')} |`,
    `| ${header.map(() => '---').join(' | ')} |`,
    ...rows.map((row) => `| ${row.join(' | ')} |`),
  ].join('\n');
}

function stripTag(image) {
  return image.replace(/@sha256:.*$/, '').replace(/:[^/]+$/, '');
}

function shippedImages() {
  const found = new Map();
  const add = (ref, via) => {
    const image = stripTag(ref);
    if (!OWN_IMAGES.test(image)) {
      found.set(image, (found.get(image) ?? new Set()).add(via));
    }
  };
  for (const file of globSync('deploy/docker/*/docker-compose.yaml')) {
    for (const [, ref] of readFileSync(file, 'utf8').matchAll(/^\s*(?:-\s+)?image:\s*['"]?([^'"\s#]+)/gm)) {
      add(ref, 'Docker Compose');
    }
  }
  const values = readFileSync('deploy/charts/distr/values.yaml', 'utf8');
  for (const [, ref] of values.matchAll(/^\s*(?:-\s+)?(?:image|repository):\s*['"]?([^'"\s#]+)/gm)) {
    add(ref, 'Helm chart');
  }
  const dockerfiles = globSync(['Dockerfile.*', '*/Dockerfile'], {exclude: ['node_modules/**']});
  for (const file of dockerfiles.sort()) {
    const stages = [...readFileSync(file, 'utf8').matchAll(/^FROM\s+(?:--\S+\s+)*(\S+)/gm)];
    add(stages.at(-1)[1], `base of \`${file}\``);
  }
  return found;
}

function shippedCharts() {
  const chart = readFileSync('deploy/charts/distr/Chart.yaml', 'utf8');
  const dependencies = chart.match(/^dependencies:\n((?:[ -].*\n?)*)/m)?.[1] ?? '';
  return new Set([...dependencies.matchAll(/^\s*-\s+name:\s*(\S+)/gm)].map(([, name]) => name));
}

function components() {
  const images = shippedImages();
  const charts = shippedCharts();
  const listedImages = new Set();
  const listedCharts = new Set();
  const rows = readJSON('components.json').map((component) => {
    const via = new Set();
    for (const image of component.images ?? []) {
      listedImages.add(image);
      images.get(image)?.forEach((v) => via.add(v));
    }
    for (const chart of component.charts ?? []) {
      listedCharts.add(chart);
      if (charts.has(chart)) {
        via.add('Helm chart');
      }
    }
    if (via.size === 0) {
      errors.push(`components.json lists "${component.name}", which is no longer shipped`);
    }
    return [
      component.name,
      component.license,
      [...via].join(', '),
      link(component.source.replace(/^https:\/\//, ''), component.source),
    ];
  });
  for (const image of images.keys()) {
    if (!listedImages.has(image)) {
      errors.push(`image ${image} is shipped but missing from components.json`);
    }
  }
  for (const chart of charts) {
    if (!listedCharts.has(chart)) {
      errors.push(`chart ${chart} is a dependency of the Helm chart but missing from components.json`);
    }
  }
  return table(['Component', 'License', 'Shipped as', 'Source'], rows);
}

function backend() {
  const overrides = readJSON('go-license-overrides.json');
  const usedOverrides = new Set();
  const template = join(mkdtempSync(join(tmpdir(), 'go-licenses-')), 'report.tmpl');
  writeFileSync(
    template,
    '{{ range . }}{{ .Name }}\t{{ .Version }}\t{{ .LicenseName }}\t{{ .LicenseURL }}\t{{ .LicensePath }}\n{{ end }}'
  );
  const report = execFileSync('go-licenses', ['report', './cmd/...', '--ignore', OWN_MODULE, '--template', template], {
    env: {...process.env, GOOS: 'linux', GOARCH: 'amd64'},
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  rmSync(dirname(template), {recursive: true});
  const modules = new Map();
  for (const line of report.split('\n').filter(Boolean)) {
    let [name, version, license, url, licensePath] = line.split('\t');
    if (license === 'Unknown') {
      const key = Object.keys(overrides).find((prefix) => name === prefix || name.startsWith(`${prefix}/`));
      if (!key) {
        errors.push(`go-licenses cannot identify the license of ${name}, add it to go-license-overrides.json`);
        continue;
      }
      if (!usedOverrides.has(key)) {
        usedOverrides.add(key);
        collectNotices(key, execFileSync('go', ['list', '-m', '-f', '{{.Dir}}', key], {encoding: 'utf8'}).trim());
      }
      ({license, url} = overrides[key]);
      name = key;
    } else {
      collectNotices(name, dirname(licensePath));
    }
    // go-licenses reports a module once per license it finds, and all of them apply.
    const module = modules.get(`${name}@${version}`) ?? {name, version, licenses: new Map()};
    module.licenses.set(license, url);
    modules.set(`${name}@${version}`, module);
  }
  for (const key of Object.keys(overrides)) {
    if (!usedOverrides.has(key)) {
      errors.push(`go-license-overrides.json lists ${key}, which go-licenses no longer needs`);
    }
  }
  const goVersion = execFileSync('go', ['env', 'GOVERSION'], {encoding: 'utf8'}).trim().replace(/^go/, '');
  const rows = [
    ['Go standard library', goVersion, link('BSD-3-Clause', 'https://go.dev/LICENSE')],
    ...[...modules.values()]
      .sort((a, b) => a.name.localeCompare(b.name))
      .map(({name, version, licenses}) => {
        const sorted = [...licenses].sort(([a], [b]) => a.localeCompare(b));
        const urls = new Set(licenses.values());
        const expression =
          urls.size === 1
            ? link(sorted.map(([license]) => license).join(' AND '), [...urls][0])
            : sorted.map(([license, url]) => link(license, url)).join(' AND ');
        return [name, version, expression];
      }),
  ];
  return table(['Module', 'Version', 'License'], rows);
}

function frontend() {
  const report = JSON.parse(execFileSync('pnpm', ['licenses', 'list', '--prod', '--json'], {encoding: 'utf8'}));
  const rows = Object.values(report)
    .flat()
    .filter(({name}) => name !== '@distr-sh/distr-sdk')
    .sort((a, b) => a.name.localeCompare(b.name))
    .map(({name, versions, license, homepage, paths}) => {
      paths.forEach((path) => collectNotices(name, path));
      if (license === 'Unknown') {
        errors.push(`the license of npm package ${name} is unknown, run "pnpm install --frozen-lockfile" first`);
      }
      return [link(name, homepage), versions.join(', '), license];
    });
  return table(['Package', 'Version', 'License'], rows);
}

function noticeSection() {
  const byText = new Map();
  for (const {name, text} of notices.values()) {
    byText.set(text, [...(byText.get(text) ?? []), name]);
  }
  return [...byText]
    .map(([text, names]) => [[...new Set(names)].sort((a, b) => a.localeCompare(b)), text])
    .sort(([a], [b]) => a[0].localeCompare(b[0]))
    .map(([names, text]) => `\n--- ${names.join(', ')}\n\n${text}\n`)
    .join('');
}

function render(template, output, sections) {
  writeFileSync(
    output,
    readFileSync(join(DIR, template), 'utf8').replace(/\{\{(\w+)\}\}/g, (_, name) => sections[name])
  );
}

const sections = {components: components(), backend: backend(), frontend: frontend()};

if (errors.length > 0) {
  console.error(errors.map((error) => `error: ${error}`).join('\n'));
  process.exit(1);
}

render('template.md', 'THIRD_PARTY_LICENSES.md', sections);
render('notice.txt', 'NOTICE', {notices: noticeSection()});
