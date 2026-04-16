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
    return git(`log --first-parent --format="%H %s" ${range}`).split('\n').filter(Boolean);
  } catch {
    return [];
  }
}

function parseCommit(line) {
  const hash = line.slice(0, 40);
  const subject = line.slice(41);
  const m = subject.match(COMMIT_RE);
  if (!m) return null;
  const entry = {
    scope: m.groups.scope,
    description: m.groups.description,
    commit: hash.slice(0, 8),
  };
  if (m.groups.pr) entry.pr = Number(m.groups.pr);
  return {type: m.groups.type, entry};
}

function buildRelease(version, lines) {
  const byType = new Map();
  for (const line of lines) {
    const parsed = parseCommit(line);
    if (!parsed) continue;
    if (!byType.has(parsed.type)) byType.set(parsed.type, []);
    byType.get(parsed.type).push(parsed.entry);
  }
  if (byType.size === 0) return null;
  return {
    version,
    types: [...byType.entries()].map(([type, changes]) => ({type, changes})),
  };
}

const outputFile = process.argv[2] || 'agent-changelog.json';
const branch = process.argv[3] || 'main';

const tags = getTags(branch);
const ranges = [
  ...tags.map((tag, i) => [tag, i === 0 ? tag : `${tags[i - 1]}..${tag}`]),
  ['unreleased', `${tags.at(-1)}..${branch}`],
];

const releases = ranges
  .reverse()
  .map(([version, range]) => buildRelease(version, getCommits(range)))
  .filter(Boolean);

writeFileSync(outputFile, JSON.stringify({releases}, null, 2) + '\n');
console.log(`Wrote agent changelog to ${outputFile} (${releases.length} releases with agent changes)`);
