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
TOKEN="{{.Token}}"

_tmpdir=$(mktemp -d)
trap 'rm -rf "$_tmpdir"' EXIT

upload_resource() {
  _name="$1"
  _content="$2"
  _tmpfile="${_tmpdir}/upload_content.tmp"
  printf '%s' "$_content" > "$_tmpfile"
  if ! curl -fsSL -X POST \
    -F "name=${_name}" \
    -F "content=@${_tmpfile}" \
    "${BASE_URL}/resources?token=${TOKEN}" > /dev/null 2>&1; then
    echo "    Warning: failed to upload ${_name}"
  fi
  rm -f "$_tmpfile"
}

echo "=== Distr Support Bundle Collector ==="
echo "Bundle ID: ${BUNDLE_ID}"
echo ""

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

EXCLUDE_SET=""
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
  EXCLUDE_INPUT=$(printf '%s' "$EXCLUDE_INPUT" | tr -d ' ')

  if [ -n "$EXCLUDE_INPUT" ]; then
    EXCLUDE_SET=",$EXCLUDE_INPUT,"
  fi
fi

# Collect environment variables from host and containers
echo ""
echo "Collecting environment variables..."
ENV_GROUP_COUNT=0

# Collect host environment variables
ENV_GROUP_COUNT=$((ENV_GROUP_COUNT + 1))
HOST_ENV=""
{{- range .EnvVars}}
_val=$(printenv "{{.Name}}" 2>/dev/null || true)
{{- if .Redacted}}
if [ -n "$_val" ]; then _val="[REDACTED]"; fi
{{- end}}
HOST_ENV="${HOST_ENV}{{.Name}}=${_val}
"
{{- end}}
printf '%s' "$HOST_ENV" > "${_tmpdir}/envgroup_${ENV_GROUP_COUNT}.txt"
printf '%s' "Host" > "${_tmpdir}/envgroup_${ENV_GROUP_COUNT}.name"
printf '%s' "host-environment-variables" > "${_tmpdir}/envgroup_${ENV_GROUP_COUNT}.resource"

# Collect container environment variables
if [ -n "$CONTAINERS" ]; then
  IDX=1
  while IFS="$(printf '\t')" read -r CID CNAME CSTATUS _CIMAGE; do
    if [ -n "$EXCLUDE_SET" ] && echo "$EXCLUDE_SET" | grep -q ",$IDX,"; then
      IDX=$((IDX + 1))
      continue
    fi

    ENV_GROUP_COUNT=$((ENV_GROUP_COUNT + 1))
    CONTAINER_ENV=$(docker exec "$CID" env 2>/dev/null)
    _exec_rc=$?
    if [ $_exec_rc -eq 0 ]; then
      FILTERED_ENV=""
{{- range .EnvVars}}
      _val=$(echo "$CONTAINER_ENV" | grep "^{{.Name}}=" | head -1 | cut -d= -f2-)
{{- if .Redacted}}
      if [ -n "$_val" ]; then _val="[REDACTED]"; fi
{{- end}}
      FILTERED_ENV="${FILTERED_ENV}{{.Name}}=${_val}
"
{{- end}}
      printf '%s' "$FILTERED_ENV" > "${_tmpdir}/envgroup_${ENV_GROUP_COUNT}.txt"
    else
      printf '%s' "Error: could not collect environment variables (container may be stopped)" > "${_tmpdir}/envgroup_${ENV_GROUP_COUNT}.txt"
    fi
    printf '%s' "$CNAME" > "${_tmpdir}/envgroup_${ENV_GROUP_COUNT}.name"
    printf '%s' "${CNAME}-container-env" > "${_tmpdir}/envgroup_${ENV_GROUP_COUNT}.resource"

    IDX=$((IDX + 1))
  done <<EOF_CONTAINERS
$CONTAINERS
EOF_CONTAINERS
fi

# Display environment variable groups and let user select
if [ "$ENV_GROUP_COUNT" -gt 0 ]; then
  echo ""
  echo "Environment variables to upload:"
  echo "---"
  _g=1
  while [ "$_g" -le "$ENV_GROUP_COUNT" ]; do
    _gname=$(cat "${_tmpdir}/envgroup_${_g}.name")
    printf "  [%d] %s\n" "$_g" "$_gname"
    while IFS= read -r _line; do
      printf "      %s\n" "$_line"
    done < "${_tmpdir}/envgroup_${_g}.txt"
    echo ""
    _g=$((_g + 1))
  done

  echo "Enter group numbers to EXCLUDE from upload (comma-separated), or press Enter to include all:"
  read -r ENV_EXCLUDE_INPUT
  ENV_EXCLUDE_INPUT=$(printf '%s' "$ENV_EXCLUDE_INPUT" | tr -d ' ')

  ENV_EXCLUDE_SET=""
  if [ -n "$ENV_EXCLUDE_INPUT" ]; then
    ENV_EXCLUDE_SET=",${ENV_EXCLUDE_INPUT},"
  fi

  # Upload non-excluded environment variable groups
  _g=1
  while [ "$_g" -le "$ENV_GROUP_COUNT" ]; do
    _gname=$(cat "${_tmpdir}/envgroup_${_g}.name")
    if [ -n "$ENV_EXCLUDE_SET" ] && echo "$ENV_EXCLUDE_SET" | grep -q ",$_g,"; then
      echo "  Skipping env vars for $_gname"
    else
      _gresource=$(cat "${_tmpdir}/envgroup_${_g}.resource")
      _gcontent=$(cat "${_tmpdir}/envgroup_${_g}.txt")
      if [ -n "$_gcontent" ]; then
        upload_resource "$_gresource" "$_gcontent"
        echo "  Uploaded env vars for $_gname"
      fi
    fi
    _g=$((_g + 1))
  done
fi

# Collect container logs
if [ -n "$CONTAINERS" ]; then
  echo ""
  echo "Collecting container logs..."
  IDX=1
  while IFS="$(printf '\t')" read -r CID CNAME CSTATUS _CIMAGE; do
    if [ -n "$EXCLUDE_SET" ] && echo "$EXCLUDE_SET" | grep -q ",$IDX,"; then
      IDX=$((IDX + 1))
      continue
    fi

    CONTAINER_LOGS=$(docker logs --tail 1000 "$CID" 2>&1 || true)
    if [ -n "$CONTAINER_LOGS" ]; then
      upload_resource "${CNAME}-container-logs" "$CONTAINER_LOGS"
      echo "  Uploaded logs for $CNAME (last 1000 lines)"
    else
      echo "  No logs available for $CNAME"
    fi

    IDX=$((IDX + 1))
  done <<EOF_CONTAINERS
$CONTAINERS
EOF_CONTAINERS
fi

# Finalize support bundle
echo ""
echo "Finalizing support bundle..."
if ! curl -fsSL -X POST "${BASE_URL}/finalize?token=${TOKEN}" > /dev/null 2>&1; then
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
