#!/bin/sh
set -e

need_tty=yes
_dir=$(mktemp -d)
_script="${_dir}/distr-collect.sh"
trap 'rm -rf "$_dir"' EXIT

# Write the collect script to a temp file. When piped (curl | sh), the script
# is run as a child process with /dev/tty as stdin so that interactive prompts
# work. This is the same pattern used by rustup-init.sh.
cat > "$_script" << 'DISTR_COLLECT_EOF'
#!/bin/sh

BUNDLE_ID="{{.BundleID}}"
BASE_URL="{{.BaseURL}}"
AUTH_HEADER="{{.AuthHeader}}"

upload_resource() {
  _name="$1"
  _content="$2"
  _json_content=$(printf '%s' "$_content" | \
    tr -d '\000-\010\013-\037' | \
    awk 'BEGIN{ORS=""} {gsub(/\\/,"\\\\"); gsub(/"/,"\\\""); gsub(/\t/,"\\t"); if(NR>1) printf "\\n"; print}')
  if ! printf '{"name":"%s","content":"%s"}' "$_name" "$_json_content" | \
    curl -fsSL -X POST \
      -H "${AUTH_HEADER}" \
      -H "Content-Type: application/json" \
      "${BASE_URL}/resources" \
      -d @- > /dev/null 2>&1; then
    echo "    Warning: failed to upload ${_name}"
  fi
}

is_redacted() {
  case "$1" in
{{- range .EnvVars}}{{if .Redacted}}
    "{{.Name}}") return 0 ;;
{{- end}}{{end}}
    *) return 1 ;;
  esac
}

echo "=== Distr Support Bundle Collector ==="
echo "Bundle ID: ${BUNDLE_ID}"
echo ""

# Collect host environment variables
echo "Collecting host environment variables..."
HOST_ENV=""
{{- range .EnvVars}}
_val=$(printenv "{{.Name}}" 2>/dev/null || true)
{{- if .Redacted}}
if [ -n "$_val" ]; then _val="[REDACTED]"; fi
{{- end}}
HOST_ENV="${HOST_ENV}{{.Name}}=${_val}
"
{{- end}}
if [ -n "$HOST_ENV" ]; then
  upload_resource "host-environment-variables" "$HOST_ENV"
  echo "  Uploaded host environment variables"
fi

# Collect system information
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

# List Docker containers
echo ""
echo "Detecting Docker containers..."
CONTAINERS=$(docker ps -a --format "{{`{{.ID}}`}}	{{`{{.Names}}`}}	{{`{{.Status}}`}}	{{`{{.Image}}`}}" 2>/dev/null || true)

if [ -z "$CONTAINERS" ]; then
  echo "  No Docker containers found (docker may not be available)"
else
  echo ""
  echo "Available containers:"
  echo "---"
  IDX=1
  while IFS="$(printf '\t')" read -r CID CNAME CSTATUS CIMAGE; do
    printf "  [%d] %s (%s) - %s\n" "$IDX" "$CNAME" "$CSTATUS" "$CIMAGE"
    IDX=$((IDX + 1))
  done <<EOF_CONTAINERS
$CONTAINERS
EOF_CONTAINERS
  echo ""
  echo "Enter container numbers to EXCLUDE (comma-separated), or press Enter to include all:"
  read -r EXCLUDE_INPUT

  # Build exclusion set
  EXCLUDE_SET=""
  if [ -n "$EXCLUDE_INPUT" ]; then
    EXCLUDE_SET=",$EXCLUDE_INPUT,"
  fi

  IDX=1
  while IFS="$(printf '\t')" read -r CID CNAME CSTATUS _CIMAGE; do
    if [ -n "$EXCLUDE_SET" ] && echo "$EXCLUDE_SET" | grep -q ",$IDX,"; then
      echo "  Skipping $CNAME"
      IDX=$((IDX + 1))
      continue
    fi

    echo "  Collecting data for $CNAME..."

    # Collect container environment variables via docker exec
    CONTAINER_ENV=$(docker exec "$CID" env 2>/dev/null || true)
    if [ -n "$CONTAINER_ENV" ]; then
      REDACTED_ENV=""
      while IFS= read -r env_line; do
        env_var_name=$(echo "$env_line" | cut -d= -f1)
        env_var_value=$(echo "$env_line" | cut -d= -f2-)
        if is_redacted "$env_var_name" && [ -n "$env_var_value" ]; then
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
  done <<EOF_CONTAINERS
$CONTAINERS
EOF_CONTAINERS
fi

# Finalize support bundle
echo ""
echo "Finalizing support bundle..."
if ! curl -fsSL -X POST -H "${AUTH_HEADER}" "${BASE_URL}/finalize" > /dev/null 2>&1; then
  echo "Warning: failed to finalize support bundle"
fi
echo ""
echo "Support bundle collection complete!"
echo "Bundle ID: ${BUNDLE_ID}"
DISTR_COLLECT_EOF

chmod u+x "$_script"

if [ "$need_tty" = "yes" ] && [ ! -t 0 ]; then
  # The script was piped into sh (e.g., curl | sh) and doesn't have stdin to
  # pass to the child process. Explicitly connect /dev/tty to stdin.
  if [ ! -t 1 ]; then
    echo "Unable to run interactively." >&2
    exit 1
  fi
  sh "$_script" < /dev/tty
else
  sh "$_script"
fi
