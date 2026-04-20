#!/usr/bin/env node

import {execSync} from 'node:child_process';
import {writeFileSync} from 'node:fs';

const COMMIT_RE =
  /^(?<type>[a-z]+)\((?<scope>kubernetes-agent|docker-agent|agent)\): (?<description>.+?)(?:\s\(#(?<pr>\d+)\))?$/;

function git(args) {
  return execSync(`git ${args}`, {encoding: 'utf-8'}).trimEnd();
}

function getTags(branch) {
  return git(`tag --sort=v:refname --merged ${branch}`)
    .split('\n')
    .filter((t) => /^\d+\.\d+\.\d+$/.test(t));
}

function getCommits(range) {
  try {
    return git(`log --first-parent --format="%h %s" ${range}`).split('\n').filter(Boolean);
  } catch {
    return [];
  }
}

function parseCommit(line) {
  const i = line.indexOf(' ');
  const hash = line.slice(0, i);
  const subject = line.slice(i + 1);
  const m = subject.match(COMMIT_RE);
  if (!m) return null;
  const entry = {
    scope: m.groups.scope,
    description: m.groups.description,
    commit: hash,
  };
  if (m.groups.pr) entry.pr = Number(m.groups.pr);
  return {section: m.groups.type, entry};
}

function buildRelease(version, lines) {
  const bySection = new Map();
  for (const line of lines) {
    const parsed = parseCommit(line);
    if (!parsed) continue;
    if (!bySection.has(parsed.section)) bySection.set(parsed.section, []);
    bySection.get(parsed.section).push(parsed.entry);
  }
  if (bySection.size === 0) return null;
  return {
    version,
    sections: [...bySection.entries()].map(([section, changes]) => ({section, changes})),
  };
}

const outputFile = process.argv[2] || 'agent-changelog.json';
const branch = process.argv[3] || 'main';
const nextVersion = process.argv[4] || 'unreleased';

const tags = getTags(branch);

if (tags.length === 0) {
  writeFileSync(outputFile, JSON.stringify({releases: []}, null, 2) + '\n');
  console.log(`Wrote agent changelog to ${outputFile} (0 releases — no tags found)`);
  process.exit(0);
}

const ranges = [
  ...tags.map((tag, i) => [tag, i === 0 ? tag : `${tags[i - 1]}..${tag}`]),
  [nextVersion, `${tags.at(-1)}..${branch}`],
];

const releases = ranges
  .reverse()
  .map(([version, range]) => buildRelease(version, getCommits(range)))
  .filter(Boolean);

writeFileSync(outputFile, JSON.stringify({releases}, null, 2) + '\n');
console.log(`Wrote agent changelog to ${outputFile} (${releases.length} releases with agent changes)`);
