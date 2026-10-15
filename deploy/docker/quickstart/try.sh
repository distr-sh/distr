#!/bin/sh
# Starts a local Distr proof of concept: the Community Edition quickstart with an admin user, an API token and a
# registry slug, ready for an AI coding agent or a person to onboard an application. It runs the published Distr
# image and never builds Distr from source.
#
#   sh deploy/docker/quickstart/try.sh                  in a Distr checkout, uses the quickstart next to this script
#   curl -fsSL https://distr.sh/try.sh -o try.sh && sh try.sh   anywhere else, downloads the release quickstart
#
#   sh try.sh [--kubernetes] [--client <name>]   checks the machine, downloads the images and starts Distr
#   sh try.sh --pull [--kubernetes]              only checks the machine and downloads the images, safe to repeat
#   sh try.sh --connect [--target <id>]          starts the Distr controller of a deployment target on this machine
#   sh try.sh --check-ports <port>...            fails when something this proof of concept did not deploy holds a port
#   sh try.sh --diagnose                         prints diagnostics with every secret redacted
#   sh try.sh --down                             removes Distr, its controllers and the applications they deployed
#
# --kubernetes also starts the minikube profile distr-poc (installing minikube, kubectl and helm when they are
# missing) and points Distr at it. Everything else lives in ~/.distr/poc (DISTR_POC_HOME), including the copy of this
# script at ~/.distr/poc/try.sh and ~/.distr/poc/compose.sh, which runs docker compose with the files, environment
# and project name Distr was started with.
#
# Distr listens on http://localhost:8080 and its registry on localhost:8585. DISTR_PORT moves Distr to another port
# when 8080 is taken. DISTR_IMAGE replaces the published Distr image with a locally built one. Later runs remember
# both until --down.
#
# The script and ~/.distr/poc/event.sh send anonymous progress events (started, distr_ready, packaged, deployed,
# completed, blocked) that carry no application, repository or personal data. Set DO_NOT_TRACK=1 or
# DISTR_TELEMETRY=0 to turn them off.
set -eu

DISTR_VERSION="4.4.2" # x-release-please-version
SCRIPT_VERSION="2"
JQ_VERSION="1.8.1"
POC_HOME="${DISTR_POC_HOME:-$HOME/.distr/poc}"
QUICKSTART="$POC_HOME/quickstart"
PROJECT="distr-poc"
CONTROLLER_PROJECT="distr"
MINIKUBE_PROFILE="distr-poc"
ADMIN_EMAIL="poc@example.com"
REGISTRY_SLUG="poc"
MIN_COMPOSE_VERSION="2.23.1"
MIN_COMPOSE_VERSION_OVERRIDE="2.24.4"
MIN_DOCKER_VERSION="25.0.0"
MIN_FREE_DISK_KB=3000000
PULL_ATTEMPTS=4

log() {
  printf '%s\n' "$*" >&2
}

usage() {
  log "usage: sh try.sh [--kubernetes] [--client <name>] | --pull | --connect [--target <id>] |"
  log "       --check-ports <port>... | --diagnose | --down"
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

# version_at_least <have> <need> compares dotted versions such as 2.39.1-desktop.1 and 2.24.4.
version_at_least() {
  have="$(printf '%s' "$1" | sed 's/^v//; s/[^0-9.].*$//')"
  [ -n "$have" ] || return 1
  printf '%s\n%s\n' "$2" "$have" | sort -t. -k1,1n -k2,2n -k3,3n -C
}

previous_setting() {
  if [ -f "$POC_HOME/poc.env" ]; then
    sed -n "s/^$1=//p" "$POC_HOME/poc.env"
  fi
}

init_poc_home() {
  mkdir -p "$POC_HOME/bin" "$POC_HOME/lib"
  chmod 700 "$POC_HOME"
  POC_ID="$(previous_setting POC_ID)"
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
    printf 'DISTR_IMAGE=%s\n' "${DISTR_IMAGE:-}"
    printf 'QUICKSTART_SOURCE=%s\n' "$QUICKSTART_SOURCE"
  } > "$POC_HOME/poc.env"
  write_event_helper
  write_links
  install_self
}

# Later commands run the copy in ~/.distr/poc, so they work from any directory and with the same DISTR_POC_HOME.
install_self() {
  if [ -n "$SCRIPT_PATH" ] && [ "$SCRIPT_PATH" != "$POC_HOME/lib/try.sh" ]; then
    cp "$SCRIPT_PATH" "$POC_HOME/lib/try.sh"
  fi
  if [ -f "$POC_HOME/lib/try.sh" ]; then
    printf "#!/bin/sh\nDISTR_POC_HOME='%s' exec sh '%s/lib/try.sh' \"\$@\"\n" "$POC_HOME" "$POC_HOME" > "$POC_HOME/try.sh"
    chmod 755 "$POC_HOME/try.sh"
  fi
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

port_in_use() {
  curl -s -o /dev/null -m 2 "http://127.0.0.1:$1/" && return 0
  [ $? -ne 7 ]
}

distr_running() {
  [ -n "$(docker ps -q --filter "label=com.docker.compose.project=$PROJECT" \
    --filter label=com.docker.compose.service=distr 2>/dev/null)" ]
}

compose() {
  sh "$POC_HOME/compose.sh" "$@"
}

install_jq() {
  command -v jq >/dev/null 2>&1 && return 0
  [ -x "$POC_HOME/bin/jq" ] && return 0
  case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    *) return 1 ;;
  esac
  case "$(uname -s)" in
    Darwin) os=macos ;;
    Linux) os=linux ;;
    *) return 1 ;;
  esac
  log "installing jq $JQ_VERSION into $POC_HOME/bin"
  curl -fsSL --retry 3 "https://github.com/jqlang/jq/releases/download/jq-$JQ_VERSION/jq-$os-$arch" \
    -o "$POC_HOME/bin/jq.tmp" && chmod 755 "$POC_HOME/bin/jq.tmp" && mv "$POC_HOME/bin/jq.tmp" "$POC_HOME/bin/jq"
}

preflight() {
  for tool in curl tar od base64 awk sed sort cksum df; do
    command -v "$tool" >/dev/null 2>&1 || fail preflight other "$tool is missing. Install it and run try.sh again."
  done
  install_jq || fail preflight other "jq is missing and could not be installed. Install jq and run try.sh again."
  command -v docker >/dev/null 2>&1 ||
    fail preflight docker_missing "Docker is not installed: install Docker Desktop or Docker Engine first."
  docker info >/dev/null 2>&1 ||
    fail preflight docker_missing "Docker is not running: start Docker Desktop or the Docker daemon and run try.sh again."
  docker_version="$(docker version --format '{{.Server.Version}}' 2>/dev/null || true)"
  version_at_least "$docker_version" "$MIN_DOCKER_VERSION" ||
    fail preflight docker_outdated "Docker Engine ${docker_version:-unknown} is too old, the quickstart needs $MIN_DOCKER_VERSION or later. Update Docker Desktop or Docker Engine (https://docs.docker.com/engine/install/)."
  docker compose version >/dev/null 2>&1 ||
    fail preflight compose_outdated "The Docker Compose plugin is missing. Install it: https://docs.docker.com/compose/install/"
  COMPOSE_VERSION="$(docker compose version --short 2>/dev/null | sed 's/^v//')"
  needed="$MIN_COMPOSE_VERSION"
  if [ "$DISTR_PORT" != 8080 ]; then
    needed="$MIN_COMPOSE_VERSION_OVERRIDE"
  fi
  require_compose "$needed"

  if ! distr_running; then
    if port_in_use "$DISTR_PORT"; then
      fail preflight port_in_use "Port $DISTR_PORT is already in use. Run try.sh again with DISTR_PORT set to a free port, for example DISTR_PORT=$((DISTR_PORT + 1)), and leave what holds port $DISTR_PORT alone."
    fi
    if port_in_use 8585; then
      fail preflight port_in_use "Port 8585 is already in use by something else. The local Distr registry needs it, so stop whatever listens there first."
    fi
  fi

  disk_dir="$(docker info --format '{{.DockerRootDir}}' 2>/dev/null || true)"
  [ -n "$disk_dir" ] && [ -d "$disk_dir" ] || disk_dir="$HOME"
  free_kb="$(df -Pk "$disk_dir" 2>/dev/null | awk 'NR == 2 { print $4 }')"
  case "$free_kb" in
    '' | *[!0-9]*) ;;
    *)
      if [ "$free_kb" -lt "$MIN_FREE_DISK_KB" ]; then
        fail preflight low_resources "Only $((free_kb / 1000)) MB of disk space are free on $disk_dir. The images need about 3 GB, free some space and run try.sh again."
      fi
      ;;
  esac

  memory="$(docker info --format '{{.MemTotal}}' 2>/dev/null || echo 0)"
  needed_memory=3500000000
  if [ "$MODE" = kubernetes ]; then
    needed_memory=7000000000
  fi
  if [ "$memory" -gt 0 ] && [ "$memory" -lt "$needed_memory" ]; then
    log "warning: Docker has $((memory / 1000000000)) GB of memory, Distr needs about $((needed_memory / 1000000000)) GB in $MODE mode."
  fi
}

# require_compose <version> stops before any image is downloaded when Docker Compose is too old for the quickstart.
require_compose() {
  version_at_least "$COMPOSE_VERSION" "$1" ||
    fail preflight compose_outdated "Docker Compose ${COMPOSE_VERSION:-unknown} is too old for this quickstart, which needs $1 or later. Update Docker Desktop, or on Linux the docker-compose-plugin package (https://docs.docker.com/compose/install/linux/), then run try.sh again."
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

# A Distr checkout brings its quickstart next to this script, so it is used instead of a download.
resolve_quickstart_source() {
  QUICKSTART_SOURCE="${DISTR_QUICKSTART_DIR:-}"
  if [ -z "$QUICKSTART_SOURCE" ] && [ -n "$SCRIPT_PATH" ]; then
    script_dir="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"
    if [ -f "$script_dir/docker-compose.yaml" ] && [ -f "$script_dir/.env" ]; then
      QUICKSTART_SOURCE="$script_dir"
    fi
  fi
  if [ -z "$QUICKSTART_SOURCE" ]; then
    QUICKSTART_SOURCE="$(previous_setting QUICKSTART_SOURCE)"
    case "$QUICKSTART_SOURCE" in
      release-*) ;;
      *) [ -n "$QUICKSTART_SOURCE" ] && [ -f "$QUICKSTART_SOURCE/docker-compose.yaml" ] || QUICKSTART_SOURCE="" ;;
    esac
  fi
  if [ -z "$QUICKSTART_SOURCE" ]; then
    QUICKSTART_SOURCE="release-$DISTR_VERSION"
  fi
}

prepare_quickstart() {
  mkdir -p "$QUICKSTART"
  first_run=false
  [ -f "$QUICKSTART/.env" ] || first_run=true
  case "$QUICKSTART_SOURCE" in
    release-*)
      if [ ! -f "$QUICKSTART/docker-compose.yaml" ] || [ "$(cat "$QUICKSTART/.release" 2>/dev/null)" != "$QUICKSTART_SOURCE" ]; then
        log "downloading the Distr $DISTR_VERSION quickstart"
        download="$(mktemp -d)"
        curl -fsSL --retry 3 "https://github.com/distr-sh/distr/releases/download/$DISTR_VERSION/deploy-quickstart.tar" |
          tar -x -C "$download" || fail preflight other "Could not download the Distr quickstart, check the network and run try.sh again."
        cp "$download/docker-compose.yaml" "$QUICKSTART/"
        [ "$first_run" = false ] || cp "$download/.env" "$QUICKSTART/"
        rm -rf "$download"
        printf '%s' "$QUICKSTART_SOURCE" > "$QUICKSTART/.release"
      fi
      ;;
    *)
      cp "$QUICKSTART_SOURCE/docker-compose.yaml" "$QUICKSTART/"
      [ "$first_run" = false ] || cp "$QUICKSTART_SOURCE/.env" "$QUICKSTART/"
      rm -f "$QUICKSTART/.release"
      ;;
  esac
  if [ "$first_run" = true ]; then
    set_env JWT_SECRET "$(random_base64)"
    set_env DATABASE_ENCRYPTION_KEY "$(random_base64)"
    set_env POSTGRES_PASSWORD "$(random_hex 16)"
    set_env RUSTFS_SECRET_KEY "$(random_hex 16)"
  fi
  if grep -q 'pre_start:' "$QUICKSTART/docker-compose.yaml"; then
    require_compose 5.3.0
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
  write_compose_wrapper

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

# Every later docker compose command for Distr, including the teardown, goes through this file, so start and stop
# always use the same project name, compose files and environment file.
write_compose_wrapper() {
  files="-f '$QUICKSTART/docker-compose.yaml'"
  if [ -f "$QUICKSTART/compose.override.yaml" ]; then
    files="$files -f '$QUICKSTART/compose.override.yaml'"
  fi
  cat > "$POC_HOME/compose.sh" <<EOF
#!/bin/sh
# Runs docker compose for the Distr proof of concept, for example: sh $POC_HOME/compose.sh ps
exec docker compose -p '$PROJECT' --project-directory '$QUICKSTART' --env-file '$QUICKSTART/.env' $files "\$@"
EOF
  chmod 755 "$POC_HOME/compose.sh"
}

controller_images() {
  if [ "$MODE" = docker ] && [ -z "${DISTR_IMAGE:-}" ]; then
    printf 'ghcr.io/distr-sh/distr/docker-controller:%s\n' "$DISTR_VERSION"
    printf 'ghcr.io/distr-sh/distr/docker-autoheal:%s\n' "$DISTR_VERSION"
  fi
}

# Downloads every image before anything starts. docker pull keeps the layers it finished, so running the same
# command again after an interruption or a timeout resumes the download.
pull_images() {
  images="$(compose config --images)" || fail pull other "Could not read the images of the quickstart, see the output above."
  images="$(printf '%s\n%s\n' "$images" "$(controller_images)" | grep . | sort -u)"
  total="$(printf '%s\n' "$images" | grep -c .)"
  missing=0
  for image in $images; do
    docker image inspect "$image" >/dev/null 2>&1 || missing=$((missing + 1))
  done
  if [ "$missing" -gt 0 ]; then
    log "downloading $missing of $total images. A cold start downloads about 1.5 GB and can take several minutes."
    log "If this command is interrupted or times out, run it again: finished layers are kept and the download resumes."
  fi
  index=0
  for image in $images; do
    index=$((index + 1))
    if docker image inspect "$image" >/dev/null 2>&1; then
      log "[$index/$total] $image is already downloaded"
      continue
    fi
    attempt=1
    until log "[$index/$total] downloading $image (attempt $attempt of $PULL_ATTEMPTS)" && docker pull "$image" >&2; do
      [ "$attempt" -lt "$PULL_ATTEMPTS" ] ||
        fail pull image_pull_failed "Could not download $image after $PULL_ATTEMPTS attempts. Check the network and run the same command again, the finished layers are kept."
      sleep $((attempt * 5))
      attempt=$((attempt + 1))
    done
  done
}

start_distr() {
  log "starting Distr $DISTR_VERSION as the Docker Compose project $PROJECT"
  existed="$(compose ps -aq distr 2>/dev/null || true)"
  compose up -d --remove-orphans --pull never >&2 || fail start_distr distr_not_ready "docker compose up failed, see the output above."
  # Recreating a container that was just created would kill its first database migration halfway.
  if [ -n "$existed" ] && [ "$ENV_CHANGED" = true ]; then
    compose up -d --force-recreate --no-deps --pull never distr >&2 || fail start_distr distr_not_ready "Could not restart Distr."
  fi
  attempts=0
  until curl -fsS -m 3 -o /dev/null "$DISTR_URL/ready"; do
    attempts=$((attempts + 1))
    [ "$attempts" -ge 90 ] &&
      fail start_distr distr_not_ready "Distr did not become ready, check: sh $POC_HOME/compose.sh logs --tail 100 distr"
    [ $((attempts % 10)) -eq 0 ] && log "waiting for Distr to become ready ($((attempts * 2)) seconds)"
    sleep 2
  done
}

api() {
  curl -fsS -m 20 -H "Authorization: AccessToken $(cat "$POC_HOME/token")" "$@"
}

bootstrap_admin() {
  if ! { [ -f "$POC_HOME/token" ] && api -o /dev/null "$API/context" 2>/dev/null; }; then
    if [ -f "$POC_HOME/credentials" ]; then
      # shellcheck disable=SC1091
      . "$POC_HOME/credentials"
      jwt="$(jq -n --arg email "$ADMIN_EMAIL" --arg password "$PASSWORD" '{email: $email, password: $password}' |
        curl -fsS -X POST "$API/auth/login" -H 'Content-Type: application/json' --data-binary @- | jq -r '.token // empty')" || true
    else
      PASSWORD="$(random_hex 12)"
      printf 'EMAIL=%s\nPASSWORD=%s\n' "$ADMIN_EMAIL" "$PASSWORD" > "$POC_HOME/credentials"
      chmod 600 "$POC_HOME/credentials"
      jwt="$(jq -n --arg email "$ADMIN_EMAIL" --arg password "$PASSWORD" \
        '{email: $email, password: $password, name: "POC Admin", organizationName: "Distr POC"}' |
        curl -fsS -X POST "$API/auth/register" -H 'Content-Type: application/json' --data-binary @- |
        jq -r '.token // empty')" || true
    fi
    [ -n "${jwt:-}" ] || fail bootstrap distr_not_ready "Could not create the admin user, run sh $POC_HOME/try.sh --down to start over."
    token="$(curl -fsS -X POST "$API/settings/tokens" -H "Authorization: Bearer $jwt" -H 'Content-Type: application/json' \
      -d '{"label":"distr-poc"}' | jq -r '.key // empty')" || true
    [ -n "$token" ] || fail bootstrap distr_not_ready "Could not create an API token."
    (umask 077 && printf '%s' "$token" > "$POC_HOME/token")
  fi
  # Runs on every start, so a bootstrap interrupted after the token was saved still gets its registry slug.
  organization="$(api "$API/organization")" || fail bootstrap distr_not_ready "Could not read the organization."
  slug="$(printf '%s' "$organization" | jq -r '.slug // empty')"
  if [ -z "$slug" ]; then
    printf '%s' "$organization" | jq --arg slug "$REGISTRY_SLUG" '.slug = $slug' |
      api -o /dev/null -X PUT "$API/organization" -H 'Content-Type: application/json' --data-binary @- ||
      fail bootstrap distr_not_ready "Could not set the registry slug."
    slug="$REGISTRY_SLUG"
  fi
  REGISTRY_SLUG="$slug"
}

# Coding agents often run every command in a fresh shell, so the API settings and the IDs they create live in files.
write_api_env() {
  cat > "$POC_HOME/api.env" <<EOF
DISTR_URL='$DISTR_URL'
DISTR_API='$API'
DISTR_TOKEN="\$(cat '$POC_HOME/token')"
PATH='$POC_HOME/bin':"\$PATH"
export PATH
if [ -f '$POC_HOME/ids.env' ]; then . '$POC_HOME/ids.env'; fi
EOF
}

print_summary() {
  deployment_registry="localhost:8585"
  kube_context=""
  if [ "$MODE" = kubernetes ]; then
    deployment_registry="host.minikube.internal:8585"
    kube_context="$MINIKUBE_PROFILE"
  fi
  distr_image="${DISTR_IMAGE:-$(compose config --images 2>/dev/null | grep -m 1 'distr-sh/distr-' || true)}"
  jq -n \
    --arg distrUrl "$DISTR_URL" --arg apiBase "$API" --arg pocHome "$POC_HOME" \
    --arg email "$ADMIN_EMAIL" --arg deploymentRegistry "$deployment_registry" --arg slug "$REGISTRY_SLUG" \
    --arg mode "$MODE" --arg project "$PROJECT" --arg kubeContext "$kube_context" --arg pocId "$POC_ID" \
    --arg version "$DISTR_VERSION" --arg image "$distr_image" --arg source "$QUICKSTART_SOURCE" \
    --arg enterprise "$(onboarding_link enterprise-license try-sh)" --arg cloud "$(onboarding_link cloud-trial try-sh)" \
    '{
      distrUrl: $distrUrl,
      apiBase: $apiBase,
      apiEnv: "\($pocHome)/api.env",
      tokenFile: "\($pocHome)/token",
      login: {email: $email, passwordFile: "\($pocHome)/credentials"},
      registryPushHost: "localhost:8585",
      registryHostForDeployments: $deploymentRegistry,
      registrySlug: $slug,
      mode: $mode,
      composeProject: $project,
      kubeContext: $kubeContext,
      toolsDir: "\($pocHome)/bin",
      eventHelper: "\($pocHome)/event.sh",
      pocId: $pocId,
      distrVersion: $version,
      distrImage: $image,
      quickstartSource: $source,
      commands: {
        start: "sh \($pocHome)/compose.sh up -d --pull never",
        ready: "curl -fsS \($distrUrl)/ready",
        logs: "sh \($pocHome)/compose.sh logs --tail 200 distr",
        stop: "sh \($pocHome)/compose.sh stop",
        removeDistr: "sh \($pocHome)/compose.sh down -v --remove-orphans",
        removeEverything: "sh \($pocHome)/try.sh --down",
        connectController: "sh \($pocHome)/try.sh --connect",
        diagnose: "sh \($pocHome)/try.sh --diagnose"
      },
      nextStepLinks: "\($pocHome)/links.env",
      nextSteps: {enterpriseLicense: $enterprise, cloudTrial: $cloud}
    }' > "$POC_HOME/summary.json"
  log ""
  log "Distr is running at $DISTR_URL. Log in as $ADMIN_EMAIL, the password is in $POC_HOME/credentials."
  log "This summary is also in $POC_HOME/summary.json. Remove everything with: sh $POC_HOME/try.sh --down"
  cat "$POC_HOME/summary.json"
}

target_id_of_container() {
  docker inspect "$1" --format '{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null | sed -n 's/^DISTR_TARGET_ID=//p'
}

controller_containers() {
  docker ps -aq --filter label=com.docker.compose.service=controller
  docker ps -aq --filter label=com.docker.compose.service=agent
}

# The targets whose controllers belong to this proof of concept: those --connect recorded, which works while Distr is
# down, and those Distr lists.
owned_targets() {
  cat "$POC_HOME/controllers" 2>/dev/null || true
  if [ -f "$POC_HOME/token" ]; then
    api -m 10 "$API/deployment-targets" 2>/dev/null | jq -r '.[].id' 2>/dev/null || true
  fi
}

ensure_api() {
  [ -f "$POC_HOME/token" ] && [ -f "$POC_HOME/api.env" ] || {
    log "error: there is no Distr proof of concept in $POC_HOME yet, run try.sh first."
    exit 1
  }
  curl -fsS -m 5 -o /dev/null "$DISTR_URL/ready" ||
    fail connect distr_not_ready "Distr does not answer at $DISTR_URL. Start it with: sh $POC_HOME/try.sh"
}

# Connects the controller of a deployment target on this machine without the connect URL, which contains the target
# secret, ever appearing in a terminal, a log or a file.
connect_controller() {
  ensure_api
  if [ -z "$TARGET" ] && [ -f "$POC_HOME/ids.env" ]; then
    TARGET="$(sed -n 's/^TARGET_ID=//p' "$POC_HOME/ids.env" | tail -n 1)"
  fi
  [ -n "$TARGET" ] || {
    log "error: no deployment target: pass --target <id> or save TARGET_ID=<id> to $POC_HOME/ids.env."
    exit 2
  }
  target="$(api "$API/deployment-targets/$TARGET")" ||
    fail connect other "Deployment target $TARGET does not exist in this Distr."
  type="$(printf '%s' "$target" | jq -r .type)"
  owned="$(owned_targets)"

  if [ "$type" = docker ]; then
    for container in $(docker ps -aq --filter "label=com.docker.compose.project=$CONTROLLER_PROJECT"); do
      other="$(target_id_of_container "$container")"
      [ -n "$other" ] || continue
      if [ "$other" = "$TARGET" ] && [ -n "$(docker ps -q --filter "id=$container")" ]; then
        log "The Distr controller of target $TARGET already runs on this machine."
        wait_for_controller
        return 0
      fi
      if ! printf '%s\n' "$owned" | grep -qx "$other"; then
        fail connect controller_conflict "A Distr controller of another installation (target $other) already runs as the Docker Compose project $CONTROLLER_PROJECT on this machine. Connecting would replace it and remove its applications, so stop here and ask the user."
      fi
    done
  fi

  access="$(api -X POST "$API/deployment-targets/$TARGET/access-request")" ||
    fail connect controller_not_connecting "Could not request the connect URL of target $TARGET."
  connect_url="$(printf '%s' "$access" | jq -r .connectUrl)"
  unset access
  printf '%s\n' "$TARGET" >> "$POC_HOME/controllers"

  # Distr releases up to 4.4.2 only set DISTR_REGISTRY_PLAIN_HTTP in development builds, so they tell controllers to
  # use TLS against the plain HTTP registry of this proof of concept. Later releases already render true here.
  if [ "$type" = docker ]; then
    log "starting the Distr controller of target $TARGET as the Docker Compose project $CONTROLLER_PROJECT"
    curl -fsSL "$connect_url" | sed "s/DISTR_REGISTRY_PLAIN_HTTP: 'false'/DISTR_REGISTRY_PLAIN_HTTP: 'true'/" |
      (cd "$POC_HOME" && docker compose -f - up -d) >&2 ||
      fail connect controller_not_connecting "Could not start the Distr controller, see the output above."
  else
    namespace="$(printf '%s' "$target" | jq -r '.namespace // empty')"
    kubectl --context "$MINIKUBE_PROFILE" get namespace "$namespace" >/dev/null 2>&1 ||
      kubectl --context "$MINIKUBE_PROFILE" create namespace "$namespace" >&2
    # The URL names host.minikube.internal, which only the cluster resolves.
    log "installing the Distr controller of target $TARGET in the namespace $namespace of $MINIKUBE_PROFILE"
    curl -fsSL "$(printf '%s' "$connect_url" | sed 's#//host.minikube.internal:#//localhost:#')" |
      sed 's/DISTR_REGISTRY_PLAIN_HTTP: "false"/DISTR_REGISTRY_PLAIN_HTTP: "true"/' |
      kubectl --context "$MINIKUBE_PROFILE" apply -n "$namespace" -f - >&2 ||
      fail connect controller_not_connecting "Could not install the Distr controller, see the output above."
  fi
  unset connect_url
  wait_for_controller
}

wait_for_controller() {
  attempts=0
  until [ -n "$(api "$API/deployment-targets/$TARGET" | jq -r '.reportedControllerVersionId // empty')" ]; do
    attempts=$((attempts + 1))
    [ "$attempts" -ge 60 ] &&
      fail connect controller_not_connecting "The Distr controller did not connect within 3 minutes, run sh $POC_HOME/try.sh --diagnose."
    [ $((attempts % 10)) -eq 0 ] && log "waiting for the Distr controller to connect ($((attempts * 3)) seconds)"
    sleep 3
  done
  log "The Distr controller of target $TARGET is connected."
}

check_ports() {
  busy=""
  ours=""
  for container in $(owned_controller_containers); do
    ours="$ours
$(app_projects_of_controller "$container")"
  done
  for port in "$@"; do
    port_in_use "$port" || continue
    holder="$(docker ps --filter "publish=$port" \
      --format '{{index .Labels "com.docker.compose.project"}}' 2>/dev/null | head -n 1)"
    if [ -n "$holder" ] && printf '%s\n' "$ours" | grep -qx "$holder"; then
      log "port $port: held by $holder, which this proof of concept deployed"
      continue
    fi
    busy="$busy $port"
  done
  if [ -n "$busy" ]; then
    log "error: in use:$busy"
    exit 1
  fi
  log "free: $*"
}

# redact replaces the API token, the admin password and every controller's target secret with [REDACTED].
redact() {
  secrets=""
  [ -f "$POC_HOME/token" ] && secrets="$(cat "$POC_HOME/token")"
  [ -f "$POC_HOME/credentials" ] && secrets="$secrets $(sed -n 's/^PASSWORD=//p' "$POC_HOME/credentials")"
  for container in $(controller_containers); do
    secrets="$secrets $(docker inspect "$container" --format '{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null |
      sed -n 's/^DISTR_TARGET_SECRET=//p')"
  done
  awk -v secrets="$secrets" '
    BEGIN { n = split(secrets, list, " ") }
    {
      for (i = 1; i <= n; i++) {
        if (list[i] == "") continue
        while ((at = index($0, list[i])) > 0) $0 = substr($0, 1, at - 1) "[REDACTED]" substr($0, at + length(list[i]))
      }
      print
    }' | sed -E \
    -e 's/(targetSecret=)[^&"[:space:]]+/\1[REDACTED]/g' \
    -e 's/(DISTR_TARGET_SECRET[=:] *)[^[:space:]]+/\1[REDACTED]/g' \
    -e 's/eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}/[REDACTED]/g' \
    -e 's/distr-[0-9A-Za-z]{10,}_[0-9A-Za-z]{20,}/[REDACTED]/g'
}

diagnose() {
  {
    echo "try.sh $SCRIPT_VERSION, Distr $DISTR_VERSION, $(uname -sm)"
    echo "Docker $(docker version --format '{{.Server.Version}}' 2>&1), Compose $(docker compose version --short 2>&1)"
    echo "Distr URL: $DISTR_URL, ready: $(curl -fsS -m 3 "$DISTR_URL/ready" >/dev/null 2>&1 && echo yes || echo no)"
    echo
    echo "## Distr containers"
    compose ps -a 2>&1 || true
    echo
    echo "## Distr logs"
    compose logs --tail 50 --no-color distr 2>&1 || true
    for container in $(controller_containers); do
      echo
      echo "## Controller $(docker inspect "$container" --format '{{.Name}} {{.State.Status}}' 2>&1)"
      docker logs --tail 50 "$container" 2>&1 || true
    done
  } | redact
}

compose_down() {
  (cd / && docker compose -p "$1" down -v --remove-orphans) >&2
}

# A Docker controller and the applications it deployed run in Compose projects of their own. Targets created before
# the controller rename keep the compose service name agent.
owned_controller_containers() {
  owned="$(owned_targets)"
  for container in $(controller_containers); do
    target_id="$(target_id_of_container "$container")"
    [ -n "$target_id" ] || continue
    printf '%s\n' "$owned" | grep -qx "$target_id" && echo "$container"
  done
  return 0
}

app_projects_of_controller() {
  # docker cp also reads the scratch volume of a stopped controller, which docker exec cannot.
  docker cp "$1:/scratch/deployments" - 2>/dev/null | tar -xOf - 2>/dev/null |
    grep -o '"projectName": *"[^"]*"' | sed 's/.*"\([^"]*\)"$/\1/'
}

remove_docker_controllers() {
  for container in $(owned_controller_containers); do
    for project in $(app_projects_of_controller "$container"); do
      compose_down "$project" || teardown_failed="$teardown_failed application:$project"
    done
    project="$(docker inspect "$container" --format '{{index .Config.Labels "com.docker.compose.project"}}')"
    compose_down "$project" || teardown_failed="$teardown_failed controller:$project"
  done
}

# The state in ~/.distr/poc is what identifies the controllers and applications of this proof of concept, so it is
# only deleted once everything else is gone.
teardown() {
  if [ ! -f "$POC_HOME/poc.env" ]; then
    log "Nothing to do: $POC_HOME was not created by try.sh."
    return 0
  fi
  teardown_failed=""
  if command -v docker >/dev/null 2>&1; then
    docker info >/dev/null 2>&1 || {
      log "error: Docker is not running. Start it and run sh $POC_HOME/try.sh --down again."
      exit 1
    }
    remove_docker_controllers
    if [ -f "$POC_HOME/compose.sh" ]; then
      compose down -v --remove-orphans >&2 || compose_down "$PROJECT" || teardown_failed="$teardown_failed distr"
    else
      compose_down "$PROJECT" || teardown_failed="$teardown_failed distr"
    fi
    docker logout localhost:8585 >/dev/null 2>&1 || true
  fi
  if command -v helm >/dev/null 2>&1; then
    helm registry logout localhost:8585 >/dev/null 2>&1 || true
  fi
  if command -v minikube >/dev/null 2>&1 && minikube profile list 2>/dev/null | grep -q "$MINIKUBE_PROFILE"; then
    minikube delete -p "$MINIKUBE_PROFILE" >&2 || teardown_failed="$teardown_failed minikube"
  fi
  if [ -n "$teardown_failed" ]; then
    log "error: could not remove:$teardown_failed. $POC_HOME is kept, fix the error above and run sh $POC_HOME/try.sh --down again."
    exit 1
  fi
  rm -rf "$POC_HOME"
  log "Removed the Distr proof of concept."
}

ACTION=up
MODE=docker
CLIENT=""
TARGET=""
ENV_CHANGED=false
while [ $# -gt 0 ]; do
  case "$1" in
    --kubernetes) MODE=kubernetes ;;
    --client)
      CLIENT="${2:-}"
      shift
      ;;
    --target)
      TARGET="${2:-}"
      shift
      ;;
    --pull) ACTION=pull ;;
    --connect) ACTION=connect ;;
    --diagnose) ACTION=diagnose ;;
    --down) ACTION=down ;;
    --check-ports)
      shift
      [ $# -gt 0 ] || {
        usage
        exit 2
      }
      PORTS="$*"
      ACTION=check_ports
      break
      ;;
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
[ -n "$CLIENT" ] || CLIENT="$(previous_setting CLIENT)"
[ -n "$CLIENT" ] || CLIENT="$(detect_client)"
SCRIPT_PATH=""
if [ -f "$0" ]; then
  SCRIPT_PATH="$(cd "$(dirname "$0")" && pwd)/$(basename "$0")"
fi
PATH="$POC_HOME/bin:$PATH"
[ -n "${DISTR_PORT:-}" ] || DISTR_PORT="$(previous_setting DISTR_PORT)"
DISTR_PORT="${DISTR_PORT:-8080}"
[ -n "${DISTR_IMAGE:-}" ] || DISTR_IMAGE="$(previous_setting DISTR_IMAGE)"
case "$DISTR_PORT" in
  '' | *[!0-9]* | 8585)
    log "error: DISTR_PORT must be a port number other than 8585, which the registry uses."
    exit 2
    ;;
esac
DISTR_URL="http://localhost:$DISTR_PORT"
API="$DISTR_URL/api/v1"

case "$ACTION" in
  down)
    teardown
    exit 0
    ;;
  connect)
    connect_controller
    exit 0
    ;;
  diagnose)
    diagnose
    exit 0
    ;;
  check_ports)
    # shellcheck disable=SC2086 # PORTS is a list of port numbers
    check_ports $PORTS
    exit 0
    ;;
esac

started_at="$(date +%s)"
resolve_quickstart_source
init_poc_home
if [ "$ACTION" = up ]; then
  if [ "${DO_NOT_TRACK:-0}" != "1" ] && [ "${DISTR_TELEMETRY:-1}" != "0" ]; then
    log "try.sh sends anonymous progress events without application, repository or personal data. Opt out with DO_NOT_TRACK=1."
  fi
  event started "mode=$MODE"
fi
preflight
prepare_quickstart
pull_images
if [ "$ACTION" = pull ]; then
  log "All images are downloaded. Start Distr with: sh $POC_HOME/try.sh$([ "$MODE" = kubernetes ] && echo ' --kubernetes')"
  exit 0
fi
if [ "$MODE" = kubernetes ]; then
  start_minikube
fi
start_distr
bootstrap_admin
write_api_env
event distr_ready "mode=$MODE" "minutes=$((($(date +%s) - started_at + 59) / 60))"
print_summary
