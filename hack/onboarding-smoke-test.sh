#!/bin/sh
# Runs the agent onboarding end to end on this machine and tears it down again: try.sh from this checkout with the
# published Distr image (or DISTR_IMAGE), then every shell block between the hello-distr markers of
# website/public/onboard.md exactly as an agent reads them, then try.sh --down. It keeps its state in its own
# DISTR_POC_HOME and refuses to start next to an existing proof of concept, whose project names it would share.
#
#   sh hack/onboarding-smoke-test.sh    KEEP=1 leaves everything running, remove it with sh $DISTR_POC_HOME/try.sh --down
set -eu

cd "$(dirname "$0")/.."
export DISTR_POC_HOME="${DISTR_POC_HOME:-$HOME/.distr/poc-smoke-test}"
export DO_NOT_TRACK=1
POC="$DISTR_POC_HOME"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

log() {
  printf '%s\n' "$*" >&2
}

fail() {
  log "FAILED: $*"
  if [ -f "$POC/try.sh" ]; then
    log "--- diagnostics (secrets redacted)"
    sh "$POC/try.sh" --diagnose 2>&1 | tail -n 80 >&2 || true
    if [ -n "$LOGS_STEP" ]; then
      log "--- deployment logs"
      sh -eu "$LOGS_STEP" >&2 || true
    fi
    if [ "${KEEP:-0}" != 1 ]; then
      sh "$POC/try.sh" --down >&2 || true
    fi
  fi
  exit 1
}

seconds_since() {
  echo $(($(date +%s) - $1))
}

# run shows the output of a command while it runs and keeps it in $WORK/output.log for the secret scan.
run() {
  out="$WORK/current.log"
  : > "$out"
  tail -f "$out" >&2 &
  tail_pid=$!
  status=0
  "$@" > "$out" 2>&1 || status=$?
  sleep 1
  { kill "$tail_pid" && wait "$tail_pid"; } 2>/dev/null || true
  cat "$out" >> "$WORK/output.log"
  return "$status"
}

project_containers() {
  docker ps -aq --filter "label=com.docker.compose.project=$1"
}

docker info >/dev/null 2>&1 || {
  log "Docker is not running."
  exit 1
}
if [ -f "$POC/try.sh" ]; then
  log "Removing the proof of concept a previous smoke test left in $POC"
  sh "$POC/try.sh" --down
fi
if [ -n "$(project_containers distr-poc)$(project_containers distr)" ]; then
  log "A Distr proof of concept or Distr controller is already running on this machine (Compose project distr-poc or"
  log "distr). Remove it first, for example with sh ~/.distr/poc/try.sh --down."
  exit 1
fi

if [ -z "${DISTR_PORT:-}" ]; then
  for port in 8080 8081 8082 8083 8084 8085 8086 8087 8088 8089; do
    if sh deploy/docker/quickstart/try.sh --check-ports "$port" 2>/dev/null; then
      DISTR_PORT="$port"
      break
    fi
  done
  [ -n "${DISTR_PORT:-}" ] || fail "no free port between 8080 and 8089 for Distr"
fi
export DISTR_PORT

LOGS_STEP=""
awk -v dir="$WORK" '
  /<!-- hello-distr:start -->/ { on = 1 }
  /<!-- hello-distr:end -->/ { on = 0 }
  on && /^```sh$/ { block = 1; n++; file = sprintf("%s/step-%02d.sh", dir, n); next }
  block && /^```$/ { block = 0; close(file); next }
  block { print > file }
' website/public/onboard.md
steps="$(ls "$WORK"/step-*.sh 2>/dev/null)" || fail "no shell blocks between the hello-distr markers of onboard.md"
for step in $steps; do
  # The runbook addresses the proof of concept in its default location.
  sed -e "s#~/.distr/poc#$POC#g" -e "s#\$HOME/.distr/poc#$POC#g" "$step" > "$step.tmp" && mv "$step.tmp" "$step"
done
LOGS_STEP="$(printf '%s\n' "$steps" | tail -n 1)"
grep -q '/logs' "$LOGS_STEP" || fail "the last hello-distr block of onboard.md no longer reads the deployment logs"

docker_version="$(docker version --format '{{.Server.Version}}')"
compose_version="$(docker compose version --short)"
log "Docker $docker_version, Docker Compose $compose_version, Distr on port $DISTR_PORT, state in $POC"

started="$(date +%s)"
run sh deploy/docker/quickstart/try.sh --pull --client smoke-test || fail "try.sh --pull"
pulled="$(seconds_since "$started")"
run sh "$POC/try.sh" --client smoke-test || fail "try.sh"
ready="$(seconds_since "$started")"
image="$(jq -r .distrImage "$POC/summary.json")"
curl -fsS "http://localhost:$DISTR_PORT/ready" >/dev/null || fail "Distr is not ready"

ready_at="$(date +%s)"
healthy=""
for step in $steps; do
  [ "$step" = "$LOGS_STEP" ] && break
  log "--- $(basename "$step")"
  run sh -eu "$step" || fail "$(basename "$step") of the hello-distr sequence in onboard.md"
  if grep -q '= healthy \] && break' "$step"; then
    healthy="$(seconds_since "$ready_at")"
  fi
done

log "--- $(basename "$LOGS_STEP")"
attempt=0
while :; do
  run sh -eu "$LOGS_STEP" || fail "reading the deployment logs"
  grep -qv '^== ' "$WORK/current.log" && break
  attempt=$((attempt + 1))
  [ "$attempt" -lt 12 ] || fail "the deployment sent no logs to Distr"
  sleep 10
done

secrets="$(cat "$POC/token")
$(sed -n 's/^PASSWORD=//p' "$POC/credentials")"
for container in $(docker ps -q --filter label=com.docker.compose.project=distr); do
  secrets="$secrets
$(docker inspect "$container" --format '{{range .Config.Env}}{{println .}}{{end}}' | sed -n 's/^DISTR_TARGET_SECRET=//p')"
done
printf '%s\n' "$secrets" | while IFS= read -r secret; do
  [ -n "$secret" ] || continue
  if grep -qF "$secret" "$WORK/output.log" "$POC/summary.json" "$POC/ids.env"; then
    echo leaked
  fi
done | grep -q leaked && fail "a secret appeared in the output or in the files the runbook writes"

if [ "${KEEP:-0}" = 1 ]; then
  log "KEEP=1: everything keeps running. Remove it with: sh $POC/try.sh --down"
  teardown="skipped"
else
  teardown_started="$(date +%s)"
  sh "$POC/try.sh" --down || fail "try.sh --down"
  teardown="$(seconds_since "$teardown_started")"
  [ -z "$(project_containers distr-poc)$(project_containers distr)" ] || fail "containers are left after try.sh --down"
  [ -z "$(docker ps -q --filter publish=80)" ] || fail "Hello Distr still holds port 80 after try.sh --down"
  [ ! -e "$POC" ] || fail "$POC is left after try.sh --down"
fi

cat <<EOF

Onboarding smoke test passed
  Distr image:               $image
  Docker / Docker Compose:   $docker_version / $compose_version
  pull (try.sh --pull):      ${pulled}s
  start until ready:         ${ready}s
  ready until healthy:       ${healthy}s
  teardown:                  ${teardown}s
EOF
