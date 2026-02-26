package supportbundle

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func GenerateCollectScript(baseURL string, bundleID uuid.UUID, patKey string) string {
	var sb strings.Builder

	apiBase := fmt.Sprintf("%s/api/v1/support-bundles/%s", baseURL, bundleID.String())
	authHeader := fmt.Sprintf("Authorization: AccessToken %s", patKey)

	sb.WriteString("#!/bin/sh\n")
	sb.WriteString("set -e\n\n")

	fmt.Fprintf(&sb, "BUNDLE_ID=\"%s\"\n", bundleID.String())
	fmt.Fprintf(&sb, "BASE_URL=\"%s\"\n", apiBase)
	fmt.Fprintf(&sb, "AUTH_HEADER=\"%s\"\n\n", authHeader)

	// Helper function to upload a resource
	sb.WriteString(`upload_resource() {
  _name="$1"
  _content="$2"
  _json_content=$(printf '%s' "$_content" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g' | sed ':a;N;$!ba;s/\n/\\n/g')
  curl -fsSL -X POST \
    -H "${AUTH_HEADER}" \
    -H "Content-Type: application/json" \
    "${BASE_URL}/resources" \
    -d "{\"name\":\"${_name}\",\"content\":\"${_json_content}\"}" > /dev/null 2>&1
}
`)

	sb.WriteString("\n")
	sb.WriteString("echo \"=== Distr Support Bundle Collector ===\"\n")
	sb.WriteString("echo \"Bundle ID: ${BUNDLE_ID}\"\n")
	sb.WriteString("echo \"\"\n\n")

	// Step 1: Fetch env var config
	sb.WriteString("# Fetch environment variable configuration\n")
	sb.WriteString("echo \"Fetching configuration...\"\n")
	sb.WriteString("CONFIG=$(curl -fsSL -H \"${AUTH_HEADER}\" \"${BASE_URL}/config\")\n\n")

	// Step 2: Collect host environment variables
	sb.WriteString(`# Collect host environment variables
echo "Collecting host environment variables..."
HOST_ENV=""
for var_entry in $(echo "$CONFIG" | grep -o '"name":"[^"]*","redacted":[a-z]*' | tr -d '"'); do
  var_name=$(echo "$var_entry" | sed 's/name:\([^,]*\),redacted:.*/\1/')
  var_redacted=$(echo "$var_entry" | sed 's/.*redacted://')
  var_value=$(printenv "$var_name" 2>/dev/null || true)
  if [ "$var_redacted" = "true" ] && [ -n "$var_value" ]; then
    var_value="[REDACTED]"
  fi
  HOST_ENV="${HOST_ENV}${var_name}=${var_value}
"
done
if [ -n "$HOST_ENV" ]; then
  upload_resource "host-environment-variables" "$HOST_ENV"
  echo "  Uploaded host environment variables"
fi
`)

	sb.WriteString("\n")

	// Step 3: Collect system info
	sb.WriteString(`# Collect system information
echo "Collecting system information..."
SYSTEM_INFO="whoami: $(whoami 2>/dev/null || echo 'unknown')
uname: $(uname -a 2>/dev/null || echo 'unknown')
hostname: $(hostname 2>/dev/null || echo 'unknown')
date: $(date 2>/dev/null || echo 'unknown')
uptime: $(uptime 2>/dev/null || echo 'unknown')
df:
$(df -h 2>/dev/null || echo 'unavailable')
memory:
$(free -h 2>/dev/null || echo 'unavailable')"
upload_resource "system-info" "$SYSTEM_INFO"
echo "  Uploaded system information"
`)

	sb.WriteString("\n")

	// Step 4: Docker containers
	sb.WriteString(`# List Docker containers
echo ""
echo "Detecting Docker containers..."
CONTAINERS=$(docker ps -a --format "{{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Image}}" 2>/dev/null || true)

if [ -z "$CONTAINERS" ]; then
  echo "  No Docker containers found (docker may not be available)"
else
  echo ""
  echo "Available containers:"
  echo "---"
  IDX=1
  echo "$CONTAINERS" | while IFS="$(printf '\t')" read -r CID CNAME CSTATUS CIMAGE; do
    printf "  [%d] %s (%s) - %s\n" "$IDX" "$CNAME" "$CSTATUS" "$CIMAGE"
    IDX=$((IDX + 1))
  done
  echo ""
  echo "Enter container numbers to EXCLUDE (comma-separated), or press Enter to include all:"
  read -r EXCLUDE_INPUT

  # Build exclusion set
  EXCLUDE_SET=""
  if [ -n "$EXCLUDE_INPUT" ]; then
    EXCLUDE_SET=",$EXCLUDE_INPUT,"
  fi

  IDX=1
  echo "$CONTAINERS" | while IFS="$(printf '\t')" read -r CID CNAME CSTATUS _CIMAGE; do
    if [ -n "$EXCLUDE_SET" ] && echo "$EXCLUDE_SET" | grep -q ",$IDX,"; then
      echo "  Skipping $CNAME"
      IDX=$((IDX + 1))
      continue
    fi

    echo "  Collecting data for $CNAME..."

    # Collect container environment variables via docker exec
    CONTAINER_ENV=$(docker exec "$CID" env 2>/dev/null || true)
    if [ -n "$CONTAINER_ENV" ]; then
      # Apply redaction rules from config
      REDACTED_ENV=""
      while IFS= read -r env_line; do
        env_var_name=$(echo "$env_line" | cut -d= -f1)
        env_var_value=$(echo "$env_line" | cut -d= -f2-)
        is_redacted="false"
        if echo "$CONFIG" | grep -q "\"name\":\"${env_var_name}\",\"redacted\":true"; then
          is_redacted="true"
        fi
        if [ "$is_redacted" = "true" ] && [ -n "$env_var_value" ]; then
          REDACTED_ENV="${REDACTED_ENV}${env_var_name}=[REDACTED]
"
        else
          REDACTED_ENV="${REDACTED_ENV}${env_line}
"
        fi
      done <<EOF_ENV
$CONTAINER_ENV
EOF_ENV
      upload_resource "container-env-${CNAME}" "$REDACTED_ENV"
      echo "    Uploaded environment variables"
    else
      echo "    Could not collect environment variables (container may be stopped)"
    fi

    # Collect container logs
    CONTAINER_LOGS=$(docker logs --tail 1000 "$CID" 2>&1 || true)
    if [ -n "$CONTAINER_LOGS" ]; then
      upload_resource "container-logs-${CNAME}" "$CONTAINER_LOGS"
      echo "    Uploaded logs (last 1000 lines)"
    else
      echo "    No logs available"
    fi

    IDX=$((IDX + 1))
  done
fi
`)

	sb.WriteString("\n")

	// Step 5: Finalize
	sb.WriteString(`# Finalize support bundle
echo ""
echo "Finalizing support bundle..."
curl -fsSL -X POST -H "${AUTH_HEADER}" "${BASE_URL}/finalize" > /dev/null 2>&1
echo ""
echo "Support bundle collection complete!"
echo "Bundle ID: ${BUNDLE_ID}"
`)

	return sb.String()
}
