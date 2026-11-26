#!/usr/bin/env bash
# Builds the Distr agent skill for release <tag> into dist/skill:
#   assets/  the archives uploaded to the GitHub release of <tag>
#   plugin/  the unpacked Claude Code plugin, for validation
#   site/    the files synced to distr.sh, whose index and marketplace point at those archives
set -euo pipefail

tag="${1:?usage: hack/build-skill.sh <tag>}"

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
skill_dir="$root/skills/distr"
out="$root/dist/skill"
assets="$out/assets"
plugin="$out/plugin"
site="$out/site/.well-known/agent-skills"
release_url="https://github.com/distr-sh/distr/releases/download/$tag"

rm -rf "$out"
mkdir -p "$assets" "$plugin/.claude-plugin" "$plugin/skills" "$site/distr"

name="$(yq --front-matter=extract '.name' "$skill_dir/SKILL.md")"
description="$(yq --front-matter=extract '.description' "$skill_dir/SKILL.md")"
version="$(yq --front-matter=extract '.metadata.version' "$skill_dir/SKILL.md")"

# Entries must be plain relative paths: installers reject "./SKILL.md" and expect SKILL.md at the
# root. The fixed owner, mtime and order make the archive depend on the file contents alone.
(cd "$skill_dir" && find . -type f -printf '%P\n' | LC_ALL=C sort) |
  tar --create --format=ustar --no-recursion --owner=0 --group=0 --numeric-owner \
    --mtime='UTC 2020-01-01' --mode='u=rwX,go=rX' -C "$skill_dir" --files-from=- |
  gzip -n >"$assets/$name.tar.gz"

cp "$root/skills/claude-plugin.json" "$plugin/.claude-plugin/plugin.json"
cp -r "$skill_dir" "$plugin/skills/$name"
(cd "$plugin" && 7zz a -tzip -bso0 -bsp0 "$assets/$name-claude-plugin.zip" .claude-plugin skills)

archive_digest="$(sha256sum "$assets/$name.tar.gz" | cut -d' ' -f1)"
plugin_digest="$(sha256sum "$assets/$name-claude-plugin.zip" | cut -d' ' -f1)"

jq -n \
  --arg name "$name" \
  --arg description "$description" \
  --arg url "$release_url/$name.tar.gz" \
  --arg digest "sha256:$archive_digest" \
  '{
    "$schema": "https://schemas.agentskills.io/discovery/0.2.0/schema.json",
    skills: [{name: $name, type: "archive", description: $description, url: $url, digest: $digest}]
  }' >"$site/index.json"

jq -n \
  --arg name "$name" \
  --arg description "$description" \
  --arg version "$version" \
  --arg url "$release_url/$name-claude-plugin.zip" \
  --arg sha256 "$plugin_digest" \
  '{
    name: "distr",
    owner: {name: "Distr", url: "https://distr.sh"},
    metadata: {description: "Agent skills for Distr"},
    plugins: [{
      name: $name,
      description: $description,
      version: $version,
      source: {source: "archive", url: $url, sha256: $sha256}
    }]
  }' >"$site/marketplace.json"

cp "$skill_dir/SKILL.md" "$site/$name/SKILL.md"
