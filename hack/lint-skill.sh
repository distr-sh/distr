#!/usr/bin/env bash
# Validates skills/distr against the Agent Skills specification (https://agentskills.io/specification)
# and the output of hack/build-skill.sh against what the installers expect. gh skill publish covers
# the naming and frontmatter rules, the checks below what it leaves out.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
skill_dir="$root/skills/distr"
out="$root/dist/skill"
failed=0

fail() {
  echo "error: $*" >&2
  failed=1
}

name="$(yq --front-matter=extract '.name' "$skill_dir/SKILL.md")"
description="$(yq --front-matter=extract '.description' "$skill_dir/SKILL.md")"
version="$(yq --front-matter=extract '.metadata.version' "$skill_dir/SKILL.md")"

gh skill publish --dry-run "$root" || fail "gh skill publish rejects the skill"

((${#description} <= 1024)) || fail "description has ${#description} characters, at most 1024 are allowed"
(($(wc -l <"$skill_dir/SKILL.md") <= 500)) || fail "SKILL.md has more than 500 lines"
[[ "$version" == "$(jq -r '."."' "$root/.release-please-manifest.json")" ]] ||
  fail "metadata.version $version is not the version in .release-please-manifest.json"

while IFS= read -r file; do
  while IFS= read -r link; do
    [[ -e "$(dirname "$file")/$link" ]] || fail "${file#"$root"/} links to $link, which does not exist"
  done < <(grep -oP '\]\(\K(?!https?://|#)[^)#]+' "$file" || true)
done < <(find "$skill_dir" -name '*.md')

tar -tzf "$out/assets/$name.tar.gz" | grep -qx 'SKILL.md' || fail "$name.tar.gz has no SKILL.md at its root"
[[ "$(jq -r '.skills[0].digest' "$out/site/.well-known/agent-skills/index.json")" == \
  "sha256:$(sha256sum "$out/assets/$name.tar.gz" | cut -d' ' -f1)" ]] || fail "index.json digest does not match $name.tar.gz"

pnpm dlx skills@1.9.0 add "./skills/distr" --list
pnpm dlx @anthropic-ai/claude-code@2.1.332 plugin validate "$out/plugin"

exit "$failed"
