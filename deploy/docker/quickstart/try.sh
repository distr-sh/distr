#!/bin/sh
# Starts a local Distr proof of concept: the Community Edition quickstart with an admin user, an API token and a
# registry slug, ready for an AI coding agent or a person to onboard an application.
#
#   curl -fsSL https://distr.sh/try.sh -o try.sh && sh try.sh [--kubernetes] [--client <name>]
#   sh try.sh --down    stops the proof of concept and deletes its data
#
# --kubernetes also starts the minikube profile distr-poc (installing minikube, kubectl and helm when they are
# missing) and points Distr at it. Everything else lives in ~/.distr/poc (DISTR_POC_HOME).
#
# Distr listens on http://localhost:8080 and its registry on localhost:8585. DISTR_PORT moves Distr to another port
# when 8080 is taken; later runs and --down remember it.
#
# The script and ~/.distr/poc/event.sh send anonymous progress events (started, distr_ready, packaged, deployed,
# completed, blocked) that carry no application, repository or personal data. Set DO_NOT_TRACK=1 or
# DISTR_TELEMETRY=0 to turn them off.
set -eu

DISTR_VERSION="4.4.1" # x-release-please-version
SCRIPT_VERSION="1"
POC_HOME="${DISTR_POC_HOME:-$HOME/.distr/poc}"
QUICKSTART="$POC_HOME/quickstart"
PROJECT="distr-poc"
MINIKUBE_PROFILE="distr-poc"
ADMIN_EMAIL="poc@example.com"
REGISTRY_SLUG="poc"

log() {
  printf '%s\n' "$*" >&2
}

usage() {
  log "usage: sh try.sh [--kubernetes] [--client <name>] | sh try.sh --down"
}

event() {
  if [ -x "$POC_HOME/event.sh" ]; then
    "$POC_HOME/event.sh" "$@" || true
  fi
}

# fail <step> <blocker> <message>
fail() {
  event blocked "step=$1" "blocker=$2"
  log "error: $3"
  exit 1
}

random_hex() {
  od -An -N"$1" -tx1 /dev/urandom | tr -d ' \n'
}

random_base64() {
  head -c 32 /dev/urandom | base64 | tr -d '\n'
}

json_field() {
  sed -n "s/.*\"$1\":\"\([^\"]*\)\".*/\1/p"
}

detect_client() {
  if [ -n "${DISTR_CLIENT:-}" ]; then
    echo "$DISTR_CLIENT"
  elif [ -n "${CLAUDECODE:-}" ]; then
    echo "claude-code"
  elif [ -n "${CURSOR_AGENT:-}" ] || [ -n "${CURSOR_TRACE_ID:-}" ]; then
    echo "cursor"
  elif [ -n "${CODEX_SANDBOX:-}" ] || [ -n "${CODEX_MANAGED_BY_NPM:-}" ]; then
    echo "codex"
  elif [ -n "${GEMINI_CLI:-}" ]; then
    echo "gemini-cli"
  else
    echo "unknown"
  fi
}

sanitize() {
  printf '%s' "$1" | tr -cd 'A-Za-z0-9._-' | cut -c1-40
}

init_poc_home() {
  mkdir -p "$POC_HOME/bin"
  chmod 700 "$POC_HOME"
  POC_ID=""
  if [ -f "$POC_HOME/poc.env" ]; then
    POC_ID="$(sed -n 's/^POC_ID=//p' "$POC_HOME/poc.env")"
  fi
  if [ -z "$POC_ID" ]; then
    POC_ID="$(random_hex 4)-$(random_hex 2)-$(random_hex 2)-$(random_hex 2)-$(random_hex 6)"
  fi
  {
    printf 'POC_ID=%s\n' "$POC_ID"
    printf 'CLIENT=%s\n' "$(sanitize "$CLIENT")"
    printf 'OS=%s\n' "$(uname -s | tr '[:upper:]' '[:lower:]')"
    printf 'ARCH=%s\n' "$(uname -m)"
    printf 'DISTR_VERSION=%s\n' "$DISTR_VERSION"
    printf 'SCRIPT_VERSION=%s\n' "$SCRIPT_VERSION"
    printf 'DISTR_PORT=%s\n' "$DISTR_PORT"
  } > "$POC_HOME/poc.env"
  write_event_helper
}

write_event_helper() {
  cat > "$POC_HOME/event.sh" <<'EOF'
#!/bin/sh
# Sends one anonymous Distr proof of concept event:
#   event.sh <started|distr_ready|packaged|deployed|completed|blocked> [key=value ...]
# Only known keys are sent and values may only contain letters, digits, dots, dashes and underscores.
# Set DO_NOT_TRACK=1 or DISTR_TELEMETRY=0 to turn events off.
[ "${DO_NOT_TRACK:-0}" = "1" ] && exit 0
[ "${DISTR_TELEMETRY:-1}" = "0" ] && exit 0
. "$(dirname "$0")/poc.env"
name="${1:-}"
[ -n "$name" ] && shift
props="\"poc_id\":\"$POC_ID\",\"client\":\"$CLIENT\",\"os\":\"$OS\",\"arch\":\"$ARCH\""
props="$props,\"distr_version\":\"$DISTR_VERSION\",\"script_version\":\"$SCRIPT_VERSION\",\"\$geoip_disable\":true"
props="$props,\"\$process_person_profile\":false"
step=unknown
blocker=other
for pair in "$@"; do
  key="${pair%%=*}"
  value="${pair#*=}"
  case "$key" in
    step|blocker)
      case "$value" in ''|*[!a-z_]*) continue ;; esac
      [ "$key" = step ] && step="$value" || blocker="$value"
      ;;
    mode|packaging|built_images|generated|attempts|minutes|minutes_total|http_status|docker_version|compose_version)
      case "$value" in ''|*[!A-Za-z0-9._-]*) continue ;; esac
      case "$value" in
        true|false) props="$props,\"$key\":$value" ;;
        *[!0-9]*) props="$props,\"$key\":\"$value\"" ;;
        *) props="$props,\"$key\":$value" ;;
      esac
      ;;
  esac
done
case "$name" in
  started|distr_ready|packaged|deployed|completed)
    event="agent_poc_$name"
    ;;
  blocked)
    event='$exception'
    props="$props,\"step\":\"$step\",\"blocker\":\"$blocker\""
    props="$props,\"\$exception_list\":[{\"type\":\"DistrPocBlocked\",\"value\":\"$blocker at $step\""
    props="$props,\"mechanism\":{\"handled\":true,\"synthetic\":true}}],\"\$exception_fingerprint\":\"$step:$blocker\""
    ;;
  *)
    exit 0
    ;;
esac
curl -fsS -m 3 -X POST "${DISTR_TELEMETRY_URL:-https://edge.distr.sh}/i/v0/e/" -H 'Content-Type: application/json' \
  --data "{\"api_key\":\"phc_kT9cJBgrQwRx9ayg0PUQkukLFowbTfN47jQuWqScglR\",\"event\":\"$event\",\"distinct_id\":\"$POC_ID\",\"properties\":{$props}}" \
  >/dev/null 2>&1 || true
EOF
  chmod 755 "$POC_HOME/event.sh"
}

port_in_use() {
  curl -s -o /dev/null -m 2 "http://127.0.0.1:$1/" && return 0
  [ $? -ne 7 ]
}

version_at_least() {
  have_major="${1%%.*}"
  have_rest="${1#*.}"
  have_minor="${have_rest%%.*}"
  [ "$have_major" -gt "$2" ] || { [ "$have_major" -eq "$2" ] && [ "$have_minor" -ge "$3" ]; }
}

compose() {
  if [ -f "$QUICKSTART/compose.override.yaml" ]; then
    docker compose -p "$PROJECT" --project-directory "$QUICKSTART" -f "$QUICKSTART/docker-compose.yaml" \
      -f "$QUICKSTART/compose.override.yaml" "$@"
  else
    docker compose -p "$PROJECT" --project-directory "$QUICKSTART" -f "$QUICKSTART/docker-compose.yaml" "$@"
  fi
}

check_docker() {
  command -v docker >/dev/null 2>&1 || fail preflight docker_missing "Docker is not installed: install Docker Desktop or Docker Engine first."
  docker info >/dev/null 2>&1 || fail preflight docker_missing "Docker is not running: start Docker Desktop or the Docker daemon."
  docker compose version >/dev/null 2>&1 || fail preflight compose_outdated "The Docker Compose plugin is missing."
  memory="$(docker info --format '{{.MemTotal}}' 2>/dev/null || echo 0)"
  needed=3500000000
  if [ "$MODE" = kubernetes ]; then
    needed=7000000000
  fi
  if [ "$memory" -gt 0 ] && [ "$memory" -lt "$needed" ]; then
    log "warning: Docker has $((memory / 1000000000)) GB of memory, Distr needs about $((needed / 1000000000)) GB in $MODE mode."
  fi
  if [ -z "$(compose ps -q distr 2>/dev/null || true)" ]; then
    if port_in_use "$DISTR_PORT"; then
      fail preflight port_in_use "Port $DISTR_PORT is already in use. Run try.sh again with DISTR_PORT set to a free port, for example DISTR_PORT=$((DISTR_PORT + 1))."
    fi
    if port_in_use 8585; then
      fail preflight port_in_use "Port 8585 is already in use by something else. The local Distr registry needs it, so stop whatever listens there first."
    fi
  fi
}

install_tool() {
  tool="$1"
  command -v "$tool" >/dev/null 2>&1 && return 0
  [ -x "$POC_HOME/bin/$tool" ] && return 0
  log "installing $tool"
  if [ "$(uname -s)" = Darwin ] && command -v brew >/dev/null 2>&1; then
    case "$tool" in
      kubectl) brew install kubernetes-cli >&2 ;;
      *) brew install "$tool" >&2 ;;
    esac
    return 0
  fi
  case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    *) return 1 ;;
  esac
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$tool" in
    minikube)
      curl -fsSL "https://storage.googleapis.com/minikube/releases/latest/minikube-$os-$arch" -o "$POC_HOME/bin/minikube"
      ;;
    kubectl)
      version="$(curl -fsSL https://dl.k8s.io/release/stable.txt)"
      curl -fsSL "https://dl.k8s.io/release/$version/bin/$os/$arch/kubectl" -o "$POC_HOME/bin/kubectl"
      ;;
    helm)
      version="$(curl -fsSL https://get.helm.sh/helm-latest-version)"
      curl -fsSL "https://get.helm.sh/helm-$version-$os-$arch.tar.gz" | tar -xz -C "$POC_HOME/bin" --strip-components=1 "$os-$arch/helm"
      ;;
  esac
  chmod 755 "$POC_HOME/bin/$tool"
}

start_minikube() {
  for tool in minikube kubectl helm; do
    install_tool "$tool" || fail kubernetes minikube_failed "Could not install $tool, install it manually and run try.sh again."
  done
  if ! minikube status -p "$MINIKUBE_PROFILE" >/dev/null 2>&1; then
    log "starting the minikube profile $MINIKUBE_PROFILE"
    minikube start -p "$MINIKUBE_PROFILE" --driver=docker --cpus=2 --memory=3072 --keep-context \
      --insecure-registry=host.minikube.internal:8585 >&2 ||
      fail kubernetes minikube_failed "minikube did not start, see the output above."
  fi
}

set_env() {
  awk -v key="$1" -v value="$2" '
    index($0, key "=") == 1 { print key "=\"" value "\""; found = 1; next }
    { print }
    END { if (!found) print key "=\"" value "\"" }
  ' "$QUICKSTART/.env" > "$QUICKSTART/.env.tmp"
  mv "$QUICKSTART/.env.tmp" "$QUICKSTART/.env"
}

unset_env() {
  awk -v key="$1" 'index($0, key "=") != 1' "$QUICKSTART/.env" > "$QUICKSTART/.env.tmp"
  mv "$QUICKSTART/.env.tmp" "$QUICKSTART/.env"
}

prepare_quickstart() {
  if [ ! -f "$QUICKSTART/docker-compose.yaml" ]; then
    mkdir -p "$QUICKSTART"
    if [ -n "${DISTR_QUICKSTART_DIR:-}" ]; then
      cp "$DISTR_QUICKSTART_DIR/docker-compose.yaml" "$DISTR_QUICKSTART_DIR/.env" "$QUICKSTART/"
    else
      curl -fsSL "https://github.com/distr-sh/distr/releases/download/$DISTR_VERSION/deploy-quickstart.tar" |
        tar -x -C "$QUICKSTART" || fail preflight other "Could not download the Distr quickstart."
    fi
    set_env JWT_SECRET "$(random_base64)"
    set_env DATABASE_ENCRYPTION_KEY "$(random_base64)"
    set_env POSTGRES_PASSWORD "$(random_hex 16)"
    set_env RUSTFS_SECRET_KEY "$(random_hex 16)"
  fi
  if grep -q 'pre_start:' "$QUICKSTART/docker-compose.yaml"; then
    compose_version="$(docker compose version --short 2>/dev/null | tr -d 'v')"
    version_at_least "$compose_version" 5 3 ||
      fail preflight compose_outdated "Distr $DISTR_VERSION needs Docker Compose 5.3 or later, update Docker Desktop or the Compose plugin."
  fi
  rm -f "$QUICKSTART/compose.override.yaml"
  if [ -n "${DISTR_IMAGE:-}" ] || [ "$DISTR_PORT" != 8080 ]; then
    printf 'services:\n  distr:\n' > "$QUICKSTART/compose.override.yaml"
    if [ -n "${DISTR_IMAGE:-}" ]; then
      printf '    image: %s\n' "$DISTR_IMAGE" >> "$QUICKSTART/compose.override.yaml"
    fi
    if [ "$DISTR_PORT" != 8080 ]; then
      printf "    ports: !override\n      - '%s:8080'\n      - '8585:8585'\n" "$DISTR_PORT" >> "$QUICKSTART/compose.override.yaml"
    fi
  fi
  before="$(cksum < "$QUICKSTART/.env")"
  set_env DISTR_HOST "$DISTR_URL"
  if [ "$MODE" = kubernetes ]; then
    set_env CONTROLLER_HOST "host.minikube.internal:$DISTR_PORT"
    set_env REGISTRY_HOST "host.minikube.internal:8585"
  else
    unset_env CONTROLLER_HOST
    set_env REGISTRY_HOST "localhost:8585"
  fi
  if [ "$before" != "$(cksum < "$QUICKSTART/.env")" ]; then
    ENV_CHANGED=true
  fi
}

start_distr() {
  log "starting Distr $DISTR_VERSION as the Docker Compose project $PROJECT"
  existed="$(compose ps -aq distr 2>/dev/null || true)"
  compose up -d --remove-orphans >&2 || fail start_distr distr_not_ready "docker compose up failed, see the output above."
  # Recreating a container that was just created would kill its first database migration halfway.
  if [ -n "$existed" ] && [ "$ENV_CHANGED" = true ]; then
    compose up -d --force-recreate --no-deps distr >&2 || fail start_distr distr_not_ready "Could not restart Distr."
  fi
  attempts=0
  until curl -fsS -m 3 -o /dev/null "$DISTR_URL/ready"; do
    attempts=$((attempts + 1))
    [ "$attempts" -ge 90 ] && fail start_distr distr_not_ready "Distr did not become ready, check: docker compose -p $PROJECT logs distr"
    sleep 2
  done
}

bootstrap_admin() {
  if [ -f "$POC_HOME/token" ] &&
    curl -fsS -m 5 -o /dev/null -H "Authorization: AccessToken $(cat "$POC_HOME/token")" "$API/context"; then
    return 0
  fi
  if [ -f "$POC_HOME/credentials" ]; then
    # shellcheck disable=SC1091
    . "$POC_HOME/credentials"
    jwt="$(curl -fsS -X POST "$API/auth/login" -H 'Content-Type: application/json' \
      -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$PASSWORD\"}" | json_field token)"
  else
    PASSWORD="$(random_hex 12)"
    printf 'EMAIL=%s\nPASSWORD=%s\n' "$ADMIN_EMAIL" "$PASSWORD" > "$POC_HOME/credentials"
    chmod 600 "$POC_HOME/credentials"
    jwt="$(curl -fsS -X POST "$API/auth/register" -H 'Content-Type: application/json' \
      -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$PASSWORD\",\"name\":\"POC Admin\",\"organizationName\":\"Distr POC\"}" |
      json_field token)" || true
  fi
  [ -n "${jwt:-}" ] || fail bootstrap distr_not_ready "Could not create the admin user, run sh try.sh --down to start over."
  token="$(curl -fsS -X POST "$API/settings/tokens" -H "Authorization: Bearer $jwt" -H 'Content-Type: application/json' \
    -d '{"label":"distr-poc"}' | json_field key)"
  [ -n "$token" ] || fail bootstrap distr_not_ready "Could not create an API token."
  printf '%s' "$token" > "$POC_HOME/token"
  chmod 600 "$POC_HOME/token"
  organization="$(curl -fsS -H "Authorization: AccessToken $token" "$API/organization")" ||
    fail bootstrap distr_not_ready "Could not read the organization."
  case "$organization" in
    *'"slug":null'*)
      printf '%s' "$organization" | sed "s/\"slug\":null/\"slug\":\"$REGISTRY_SLUG\"/" |
        curl -fsS -o /dev/null -X PUT "$API/organization" -H "Authorization: AccessToken $token" \
          -H 'Content-Type: application/json' --data-binary @- ||
        fail bootstrap distr_not_ready "Could not set the registry slug."
      ;;
  esac
}

# Coding agents often run every command in a fresh shell, so the API settings and the IDs they create live in files.
write_api_env() {
  cat > "$POC_HOME/api.env" <<EOF
DISTR_API='$API'
DISTR_TOKEN="\$(cat '$POC_HOME/token')"
PATH='$POC_HOME/bin':"\$PATH"
export PATH
if [ -f '$POC_HOME/ids.env' ]; then . '$POC_HOME/ids.env'; fi
EOF
}

# onboarding_link <interest> <surface>
onboarding_link() {
  printf 'https://distr.sh/onboarding/?poc_id=%s&utm_source=self-hosted-poc&utm_medium=ai-agent' "$POC_ID"
  printf '&utm_campaign=agent-onboarding&utm_content=%s&interest=%s&client=%s' "$2" "$1" "$(sanitize "$CLIENT")"
}

write_links() {
  {
    printf "ENTERPRISE_LINK='%s'\n" "$(onboarding_link enterprise-license handoff)"
    printf "CLOUD_LINK='%s'\n" "$(onboarding_link cloud-trial handoff)"
    printf "HELP_LINK='%s'\n" "$(onboarding_link help blocked)"
  } > "$POC_HOME/links.env"
}

print_summary() {
  deployment_registry="localhost:8585"
  kube_context=""
  if [ "$MODE" = kubernetes ]; then
    deployment_registry="host.minikube.internal:8585"
    kube_context="$MINIKUBE_PROFILE"
  fi
  log ""
  log "Distr is running at $DISTR_URL. Log in as $ADMIN_EMAIL, the password is in $POC_HOME/credentials."
  log "API token: $POC_HOME/token. Stop and delete everything with: sh try.sh --down"
  cat <<EOF
{
  "distrUrl": "$DISTR_URL",
  "apiBase": "$API",
  "apiEnv": "$POC_HOME/api.env",
  "tokenFile": "$POC_HOME/token",
  "login": {"email": "$ADMIN_EMAIL", "passwordFile": "$POC_HOME/credentials"},
  "registryPushHost": "localhost:8585",
  "registryHostForDeployments": "$deployment_registry",
  "registrySlug": "$REGISTRY_SLUG",
  "mode": "$MODE",
  "composeProject": "$PROJECT",
  "kubeContext": "$kube_context",
  "toolsDir": "$POC_HOME/bin",
  "eventHelper": "$POC_HOME/event.sh",
  "pocId": "$POC_ID",
  "distrVersion": "$DISTR_VERSION",
  "nextStepLinks": "$POC_HOME/links.env",
  "nextSteps": {
    "enterpriseLicense": "$(onboarding_link enterprise-license try-sh)",
    "cloudTrial": "$(onboarding_link cloud-trial try-sh)"
  }
}
EOF
}

# Compose finds a project's containers, networks and volumes by their labels, so this works even when the compose file
# is gone or no longer parses. Running it from / keeps compose from loading a compose file in the current directory.
compose_down() {
  (cd / && docker compose -p "$1" down -v --remove-orphans) >&2 || true
}

# A Docker controller and the applications it deployed run in Compose projects of their own. A controller belongs to
# this proof of concept only if its target is one of this Distr's deployment targets. Targets created before the
# controller rename keep the compose service name agent.
remove_docker_controllers() {
  targets="$(curl -fsS -m 10 -H "Authorization: AccessToken $(cat "$POC_HOME/token" 2>/dev/null)" \
    "$DISTR_URL/api/v1/deployment-targets" 2>/dev/null)" || {
    log "Distr is not reachable, so a Distr controller from this proof of concept may keep running (docker compose -p distr down)."
    return 0
  }
  for container in $(docker ps -aq --filter label=com.docker.compose.service=controller) \
    $(docker ps -aq --filter label=com.docker.compose.service=agent); do
    target_id="$(docker inspect "$container" --format '{{range .Config.Env}}{{println .}}{{end}}' |
      sed -n 's/^DISTR_TARGET_ID=//p')"
    case "$target_id" in "") continue ;; esac
    case "$targets" in *"\"$target_id\""*) ;; *) continue ;; esac
    # docker cp also reads the scratch volume of a stopped controller, which docker exec cannot.
    for project in $(docker cp "$container:/scratch/deployments" - 2>/dev/null | tar -xOf - 2>/dev/null |
      grep -o '"projectName": *"[^"]*"' | sed 's/.*"\([^"]*\)"$/\1/'); do
      compose_down "$project"
    done
    compose_down "$(docker inspect "$container" --format '{{index .Config.Labels "com.docker.compose.project"}}')"
  done
}

teardown() {
  if [ ! -f "$POC_HOME/poc.env" ]; then
    log "Nothing to do: $POC_HOME was not created by try.sh."
    return 0
  fi
  if command -v docker >/dev/null 2>&1; then
    remove_docker_controllers
    compose_down "$PROJECT"
    docker logout localhost:8585 >/dev/null 2>&1 || true
  fi
  if command -v helm >/dev/null 2>&1; then
    helm registry logout localhost:8585 >/dev/null 2>&1 || true
  fi
  if command -v minikube >/dev/null 2>&1 && minikube profile list 2>/dev/null | grep -q "$MINIKUBE_PROFILE"; then
    minikube delete -p "$MINIKUBE_PROFILE" >&2 || true
  fi
  rm -rf "$POC_HOME"
  log "Removed the Distr proof of concept."
}

MODE=docker
CLIENT=""
DOWN=false
ENV_CHANGED=false
while [ $# -gt 0 ]; do
  case "$1" in
    --kubernetes) MODE=kubernetes ;;
    --client)
      CLIENT="${2:-}"
      shift
      ;;
    --down) DOWN=true ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      usage
      exit 2
      ;;
  esac
  shift
done
[ -n "$CLIENT" ] || CLIENT="$(detect_client)"
PATH="$POC_HOME/bin:$PATH"
if [ -z "${DISTR_PORT:-}" ] && [ -f "$POC_HOME/poc.env" ]; then
  DISTR_PORT="$(sed -n 's/^DISTR_PORT=//p' "$POC_HOME/poc.env")"
fi
DISTR_PORT="${DISTR_PORT:-8080}"
case "$DISTR_PORT" in
  '' | *[!0-9]* | 8585)
    log "error: DISTR_PORT must be a port number other than 8585, which the registry uses."
    exit 2
    ;;
esac
DISTR_URL="http://localhost:$DISTR_PORT"
API="$DISTR_URL/api/v1"

if [ "$DOWN" = true ]; then
  teardown
  exit 0
fi

started_at="$(date +%s)"
init_poc_home
if [ "${DO_NOT_TRACK:-0}" != "1" ] && [ "${DISTR_TELEMETRY:-1}" != "0" ]; then
  log "try.sh sends anonymous progress events without application, repository or personal data. Opt out with DO_NOT_TRACK=1."
fi
event started "mode=$MODE"
check_docker
if [ "$MODE" = kubernetes ]; then
  start_minikube
fi
prepare_quickstart
start_distr
bootstrap_admin
write_api_env
write_links
event distr_ready "mode=$MODE" "minutes=$((($(date +%s) - started_at + 59) / 60))"
print_summary
