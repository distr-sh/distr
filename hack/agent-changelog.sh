#!/usr/bin/env bash
set -euo pipefail

# Parse git history on main for commits scoped to (kubernetes-agent) or (docker-agent),
# determine which release tag first included each commit, and output JSON.

OUTPUT_FILE="${1:-agent-changelog.json}"
BRANCH="${2:-main}"

COMMIT_PATTERN='^([a-z]+)\((kubernetes-agent|docker-agent|agent)\): (.+)$'
PR_PATTERN='^(.+) \(#([0-9]+)\)$'

# Get all semver tags (without v prefix or pre-release) sorted ascending
mapfile -t tags < <(git tag --sort=v:refname --merged "$BRANCH" | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$')

# Build tag ranges: first..tag1, tag1..tag2, ..., lastTag..HEAD
ranges=()
range_labels=()

if [[ ${#tags[@]} -gt 0 ]]; then
  # Commits before first tag
  ranges+=("${tags[0]}")
  range_labels+=("${tags[0]}")

  for ((i = 1; i < ${#tags[@]}; i++)); do
    ranges+=("${tags[i-1]}..${tags[i]}")
    range_labels+=("${tags[i]}")
  done

  # Commits after last tag (unreleased)
  ranges+=("${tags[-1]}..$BRANCH")
  range_labels+=("unreleased")
fi

releases_json="[]"

for ((i = ${#ranges[@]} - 1; i >= 0; i--)); do
  range="${ranges[i]}"
  version="${range_labels[i]}"
  commits_json="[]"

  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    hash="${line%% *}"
    subject="${line#* }"

    if [[ "$subject" =~ $COMMIT_PATTERN ]]; then
      type="${BASH_REMATCH[1]}"
      scope="${BASH_REMATCH[2]}"
      raw_description="${BASH_REMATCH[3]}"

      # Extract PR number if present, separate from description
      pr=""
      description="$raw_description"
      if [[ "$raw_description" =~ $PR_PATTERN ]]; then
        description="${BASH_REMATCH[1]}"
        pr="${BASH_REMATCH[2]}"
      fi

      commits_json=$(echo "$commits_json" | jq -c --arg t "$type" --arg s "$scope" --arg d "$description" --arg h "${hash:0:8}" --arg p "$pr" \
        '. + [{type: $t, scope: $s, description: $d, commit: $h, pr: (if $p == "" then null else ($p | tonumber) end)}]')
    fi
  done < <(git log --first-parent --format="%H %s" "$range" 2>/dev/null || true)

  # Skip versions with no agent changes
  if [[ "$(echo "$commits_json" | jq 'length')" -eq 0 ]]; then
    continue
  fi

  # Group commits by type
  types_json=$(echo "$commits_json" | jq '
    group_by(.type) |
    map({
      type: .[0].type,
      changes: map({scope: .scope, description: .description, commit: .commit} + (if .pr then {pr: .pr} else {} end))
    })
  ')

  release_entry=$(jq -n -c --arg v "$version" --argjson t "$types_json" '{version: $v, types: $t}')
  releases_json=$(echo "$releases_json" | jq -c --argjson r "$release_entry" '. + [$r]')
done

echo "$releases_json" | jq '{releases: .}' > "$OUTPUT_FILE"
echo "Wrote agent changelog to $OUTPUT_FILE ($(echo "$releases_json" | jq 'length') releases with agent changes)"
