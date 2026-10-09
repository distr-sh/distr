#!/usr/bin/env node

import {readFileSync, writeFileSync} from 'node:fs';

// The agent scopes are what commits used before the rename to controller.
const CONTROLLER_SCOPES = new Set([
  'controller',
  'docker-controller',
  'kubernetes-controller',
  'agent',
  'docker-agent',
  'kubernetes-agent',
]);

const VERSION_RE = /^## (?:\[([^\]]+)\]\([^)]+\)|([^\s]+))\s+\(([^)]+)\)/;
const SECTION_RE = /^### (.+)$/;
const ENTRY_RE =
  /^\* \*\*(?<scope>[^:*]+):\*\* (?<description>.+?)(?:\s+\(\[#(?<pr>\d+)\]\([^)]+\)\))?\s+\(\[(?<hash>[0-9a-f]+)\]\([^)]+\)\)\s*$/;

function parseChangelog(content) {
  const releases = [];
  let currentRelease = null;
  let currentSection = null;

  for (const line of content.split('\n')) {
    const versionMatch = line.match(VERSION_RE);
    if (versionMatch) {
      currentRelease = {
        version: versionMatch[1] ?? versionMatch[2],
        sections: [],
      };
      currentSection = null;
      releases.push(currentRelease);
      continue;
    }

    if (!currentRelease) continue;

    const sectionMatch = line.match(SECTION_RE);
    if (sectionMatch) {
      currentSection = {section: sectionMatch[1].trim(), changes: []};
      currentRelease.sections.push(currentSection);
      continue;
    }

    if (!currentSection) continue;

    const entryMatch = line.match(ENTRY_RE);
    if (!entryMatch) continue;

    const {scope, description, pr, hash} = entryMatch.groups;
    if (!CONTROLLER_SCOPES.has(scope)) continue;

    const entry = {scope, description, commit: hash};
    if (pr) entry.pr = Number(pr);
    currentSection.changes.push(entry);
  }

  return releases
    .map((r) => ({
      ...r,
      sections: r.sections.filter((s) => s.changes.length > 0),
    }))
    .filter((r) => r.sections.length > 0);
}

const changelogFile = process.argv[2] || 'CHANGELOG.md';
const outputFile = process.argv[3] || 'controller-changelog.json';

const content = readFileSync(changelogFile, 'utf-8');
const releases = parseChangelog(content);

writeFileSync(outputFile, JSON.stringify({releases}, null, 2) + '\n');
console.log(`Wrote controller changelog to ${outputFile} (${releases.length} releases with controller changes)`);
