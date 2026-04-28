#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────────────────────
# OpenTofu Agent Local E2E Test
#
# Prerequisites:
#   docker compose up -d       (postgres, minio, localstack)
#   mise run serve              (Hub running on :8080/:8585)
#   oras, tofu, aws CLI installed
#
# This script:
#   1. Registers a user + org on Hub
#   2. Creates an opentofu application
#   3. Pushes the sample config as an OCI artifact
#   4. Creates a deployment target + deployment
#   5. Prints env vars to run the agent locally
# ──────────────────────────────────────────────────────────────

HUB_URL="${HUB_URL:-http://localhost:8080}"
REGISTRY_HOST="${REGISTRY_HOST:-localhost:8585}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

# ── 1. Check prerequisites ──────────────────────────────────
info "Checking prerequisites..."

for cmd in oras curl jq; do
    if ! command -v "$cmd" &>/dev/null; then
        error "$cmd is required but not found"
        exit 1
    fi
done

# Check Hub is running
if ! curl -sf "${HUB_URL}/api/v1/health" &>/dev/null; then
    # Try a simple GET
    if ! curl -sf -o /dev/null -w "%{http_code}" "${HUB_URL}" &>/dev/null; then
        error "Hub is not running at ${HUB_URL}. Start with: mise run serve"
        exit 1
    fi
fi
info "Hub is reachable at ${HUB_URL}"

# Check LocalStack
if ! curl -sf "http://localhost:4566/_localstack/health" &>/dev/null; then
    error "LocalStack is not running. Start with: docker compose up -d"
    exit 1
fi
info "LocalStack is running"

# ── 2. Register user ────────────────────────────────────────
EMAIL="opentofu-test@distr.sh"
PASSWORD="TestPassword123!"
ORG_NAME="opentofu-test-org"

info "Registering user ${EMAIL}..."
REGISTER_RESP=$(curl -sf -X POST "${HUB_URL}/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\",\"organizationName\":\"${ORG_NAME}\"}" \
    2>/dev/null) || true

# Auto-verify via Mailpit if available
if curl -sf "http://localhost:8025/api/v1/messages" &>/dev/null; then
    info "Checking Mailpit for verification email..."
    sleep 2
    VERIFY_JWT=$(curl -sf "http://localhost:8025/api/v1/messages" | python3 -c "
import sys, json, re
msgs = json.load(sys.stdin)
for m in msgs.get('messages', []):
    msg = json.loads(sys.stdin.buffer.read()) if False else None
" 2>/dev/null) || true
    # Extract JWT from latest email HTML
    MSG_ID=$(curl -sf "http://localhost:8025/api/v1/messages" | python3 -c "import sys, json; msgs=json.load(sys.stdin)['messages']; print(msgs[0]['ID'] if msgs else '')" 2>/dev/null)
    if [ -n "$MSG_ID" ]; then
        VERIFY_URL=$(curl -sf "http://localhost:8025/api/v1/message/${MSG_ID}" | python3 -c "
import sys, json, re
msg = json.load(sys.stdin)
urls = re.findall(r'https?://[^\s\"<>]+verify[^\s\"<>]+', msg.get('HTML', ''))
print(urls[0] if urls else '')
" 2>/dev/null)
        if [ -n "$VERIFY_URL" ]; then
            # Replace minikube host with localhost
            VERIFY_URL=$(echo "$VERIFY_URL" | sed 's|http://[^/]*|http://localhost:8080|')
            curl -sf "$VERIFY_URL" -o /dev/null && info "Email verified via Mailpit"
        fi
    fi
fi

# Login
info "Logging in..."
LOGIN_RESP=$(curl -sf -X POST "${HUB_URL}/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}")

TOKEN=$(echo "$LOGIN_RESP" | jq -r '.token')
if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    error "Failed to login. Response: ${LOGIN_RESP}"
    error "If email verification is required, check Mailpit at http://localhost:8025"
    exit 1
fi
info "Logged in successfully"

AUTH="Authorization: Bearer ${TOKEN}"

# ── 3. Create OpenTofu application ──────────────────────────
APP_NAME="localstack-s3-test"

info "Creating application '${APP_NAME}'..."
APP_RESP=$(curl -sf -X POST "${HUB_URL}/api/v1/applications" \
    -H "Content-Type: application/json" \
    -H "$AUTH" \
    -d "{\"name\":\"${APP_NAME}\",\"type\":\"opentofu\"}" \
    2>/dev/null) || true

# Get the app
APP=$(curl -sf "${HUB_URL}/api/v1/applications" -H "$AUTH" | jq -r ".[] | select(.name == \"${APP_NAME}\")")
APP_ID=$(echo "$APP" | jq -r '.id')

if [ -z "$APP_ID" ] || [ "$APP_ID" = "null" ]; then
    error "Failed to create/find application"
    exit 1
fi
info "Application ID: ${APP_ID}"

# ── 4. Push OCI artifact ────────────────────────────────────
ARTIFACT_TAG="v0.1.0"
ARTIFACT_REF="${REGISTRY_HOST}/${ORG_NAME}/${APP_NAME}:${ARTIFACT_TAG}"

info "Pushing OpenTofu config as OCI artifact to ${ARTIFACT_REF}..."

# Create a PAT for registry auth (Distr registry uses PAT, not user password)
info "Creating access token for registry auth..."
PAT_RESP=$(curl -sf -X POST "${HUB_URL}/api/v1/settings/tokens" \
    -H "Content-Type: application/json" \
    -H "$AUTH" \
    -d "{\"label\":\"opentofu-demo\"}")
PAT=$(echo "$PAT_RESP" | jq -r '.key // empty')
if [ -z "$PAT" ]; then
    error "Failed to create PAT. Response: ${PAT_RESP}"
    exit 1
fi
info "PAT created"

# Login to registry with PAT
echo "${PAT}" | oras login "${REGISTRY_HOST}" -u "${EMAIL}" --password-stdin --insecure 2>/dev/null

# Push main.tf as OCI artifact
pushd "${SCRIPT_DIR}" > /dev/null
oras push --insecure "${ARTIFACT_REF}" \
    "main.tf:application/vnd.distr.opentofu.config.v1.tar+gzip"
popd > /dev/null

info "OCI artifact pushed"

# ── 5. Create application version ───────────────────────────
info "Creating application version ${ARTIFACT_TAG}..."
VERSION_JSON="{\"name\":\"${ARTIFACT_TAG}\",\"tofuConfigUrl\":\"${ORG_NAME}/${APP_NAME}\",\"tofuConfigVersion\":\"${ARTIFACT_TAG}\"}"
VERSION_RESP=$(curl -sf -X POST "${HUB_URL}/api/v1/applications/${APP_ID}/versions" \
    -H "$AUTH" \
    -F "applicationversion=${VERSION_JSON}" \
    2>/dev/null) || true

info "Application version created"

# ── 6. Create deployment target ─────────────────────────────
TARGET_NAME="localstack-target"

info "Creating deployment target '${TARGET_NAME}'..."
TARGET_RESP=$(curl -sf -X POST "${HUB_URL}/api/v1/deployment-targets" \
    -H "Content-Type: application/json" \
    -H "$AUTH" \
    -d "{\"name\":\"${TARGET_NAME}\",\"type\":\"opentofu\"}")

TARGET_ID=$(echo "$TARGET_RESP" | jq -r '.id // empty')
TARGET_SECRET=$(echo "$TARGET_RESP" | jq -r '.accessKey // empty')

if [ -z "$TARGET_ID" ]; then
    # Target may already exist, try to get it
    TARGETS=$(curl -sf "${HUB_URL}/api/v1/deployment-targets" -H "$AUTH")
    TARGET_ID=$(echo "$TARGETS" | jq -r ".[] | select(.name == \"${TARGET_NAME}\") | .id")
    if [ -z "$TARGET_ID" ]; then
        error "Failed to create/find deployment target"
        exit 1
    fi
    warn "Target already exists (ID: ${TARGET_ID}). You'll need the original secret."
    warn "If you lost it, delete and recreate the target via the UI at ${HUB_URL}"
fi

info "Deployment target ID: ${TARGET_ID}"

# ── 7. Create deployment ────────────────────────────────────
info "Creating deployment..."
# Get the latest version ID
VERSION_ID=$(curl -sf "${HUB_URL}/api/v1/applications" -H "$AUTH" | jq -r ".[] | select(.id == \"${APP_ID}\") | .versions[-1].id")
info "Version ID: ${VERSION_ID}"

DEPLOY_RESP=$(curl -sf -X PUT "${HUB_URL}/api/v1/deployments" \
    -H "Content-Type: application/json" \
    -H "$AUTH" \
    -d "{
        \"applicationVersionId\":\"${VERSION_ID}\",
        \"deploymentTargetId\":\"${TARGET_ID}\",
        \"tofuVars\":{\"localstack_endpoint\":\"http://host.docker.internal:4566\",\"bucket_name\":\"distr-opentofu-test\"},
        \"tofuBackendConfig\":{}
    }" 2>/dev/null) || true

info "Deployment created"

# ── 8. Print agent run command ──────────────────────────────
echo ""
echo "════════════════════════════════════════════════════════════"
echo " E2E Setup Complete!"
echo "════════════════════════════════════════════════════════════"
echo ""
echo "Hub UI:       ${HUB_URL}"
echo "Mailpit:      http://localhost:8025"
echo "LocalStack:   http://localhost:4566"
echo "MinIO:        http://localhost:9001 (distr/distr123)"
echo ""

if [ -n "${TARGET_SECRET:-}" ]; then
    echo "Run the OpenTofu agent with:"
    echo ""
    echo "  DISTR_TARGET_ID=${TARGET_ID} \\"
    echo "  DISTR_TARGET_SECRET=${TARGET_SECRET} \\"
    echo "  DISTR_LOGIN_ENDPOINT=${HUB_URL}/api/v1/agent/login \\"
    echo "  DISTR_MANIFEST_ENDPOINT=${HUB_URL}/api/v1/agent/manifest \\"
    echo "  DISTR_RESOURCE_ENDPOINT=${HUB_URL}/api/v1/agent/resources \\"
    echo "  DISTR_STATUS_ENDPOINT=${HUB_URL}/api/v1/agent/status \\"
    echo "  DISTR_METRICS_ENDPOINT=${HUB_URL}/api/v1/agent/metrics \\"
    echo "  DISTR_LOGS_ENDPOINT=${HUB_URL}/api/v1/agent/deployment-logs \\"
    echo "  DISTR_AGENT_LOGS_ENDPOINT=${HUB_URL}/api/v1/agent/deployment-target-logs \\"
    echo "  DISTR_INTERVAL=5s \\"
    echo "  DISTR_AGENT_SCRATCH_DIR=/tmp/distr-opentofu-scratch \\"
    echo "  DISTR_TOFU_PATH=$(which tofu) \\"
    echo "  go run ./cmd/agent/opentofu/"
    echo ""
fi

echo "Verify in LocalStack after agent applies:"
echo "  aws --endpoint-url=http://localhost:4566 s3 ls"
echo ""
echo "Open the Hub UI to see deployment status:"
echo "  ${HUB_URL}"
echo ""
