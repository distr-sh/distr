#!/bin/sh

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
BUNDLE_SECRET="{{.Token}}"

_tmpdir=$(mktemp -d)
trap 'rm -rf "$_tmpdir"' EXIT

upload_resource() {
  _name="$1"
  _content="$2"
  _tmpfile="${_tmpdir}/upload_content.tmp"
  _errfile="${_tmpdir}/upload_err.tmp"
  printf '%s' "$_content" > "$_tmpfile"
  if ! curl -fsSL -X POST \
    -F "name=${_name}" \
    -F "content=@${_tmpfile}" \
    "${BASE_URL}/resources?bundleSecret=${BUNDLE_SECRET}" > /dev/null 2>"$_errfile"; then
    _err=$(cat "$_errfile" 2>/dev/null)
    if [ -n "$_err" ]; then
      echo "    Warning: failed to upload ${_name}: ${_err}"
    else
      echo "    Warning: failed to upload ${_name}"
    fi
  fi
  rm -f "$_tmpfile" "$_errfile"
}

# Parse a comma-separated list of numbers and validate each is in range 1..max.
# Outputs a validated comma set like ",1,3," for use with grep.
parse_exclude_input() {
  _input="$1"
  _max="$2"
  _input=$(printf '%s' "$_input" | tr -d ' ')
  _result=""
  if [ -z "$_input" ]; then
    return
  fi
  IFS=',' read -r _dummy <<EOF_SPLIT
$_input
EOF_SPLIT
  _remaining="$_input"
  while [ -n "$_remaining" ]; do
    _entry="${_remaining%%,*}"
    if [ "$_remaining" = "$_entry" ]; then
      _remaining=""
    else
      _remaining="${_remaining#*,}"
    fi
    case "$_entry" in
      ''|*[!0-9]*)
        echo "  Warning: ignoring invalid entry '$_entry'" >&2
        continue
        ;;
    esac
    if [ "$_entry" -lt 1 ] || [ "$_entry" -gt "$_max" ]; then
      echo "  Warning: ignoring out-of-range entry '$_entry' (valid: 1-$_max)" >&2
      continue
    fi
    _result="${_result},${_entry}"
  done
  if [ -n "$_result" ]; then
    printf '%s,' "$_result"
  fi
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

echo ""
echo "System information to upload:"
echo "---"
echo "$SYSTEM_INFO"
echo "---"
echo ""
printf "Upload system information? [Y/n]: "
read -r SYSINFO_CONFIRM
case "$SYSINFO_CONFIRM" in
  [nN]*)
    echo "  Skipping system information upload"
    ;;
  *)
    upload_resource "system-info" "$SYSTEM_INFO"
    echo "  Uploaded system information"
    ;;
esac

# Detect Docker containers and build included container list
echo ""
echo "Detecting Docker containers..."
CONTAINERS=$(docker ps -a --format "{{`{{.ID}}`}}	{{`{{.Names}}`}}	{{`{{.Status}}`}}	{{`{{.Image}}`}}" 2>/dev/null || true)

CONTAINER_COUNT=0
INCLUDED_CONTAINERS=""
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
  CONTAINER_COUNT=$((IDX - 1))
  echo ""
  echo "Enter container numbers to EXCLUDE (comma-separated), or press Enter to include all:"
  read -r EXCLUDE_INPUT
  EXCLUDE_SET=$(parse_exclude_input "$EXCLUDE_INPUT" "$CONTAINER_COUNT")

  # Build the list of included containers (ID<tab>Name per line)
  IDX=1
  while IFS="$(printf '\t')" read -r CID CNAME _CSTATUS _CIMAGE; do
    if [ -z "$EXCLUDE_SET" ] || ! echo "$EXCLUDE_SET" | grep -q ",$IDX,"; then
      INCLUDED_CONTAINERS="${INCLUDED_CONTAINERS}${CID}	${CNAME}
"
    fi
    IDX=$((IDX + 1))
  done <<EOF_CONTAINERS
$CONTAINERS
EOF_CONTAINERS
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
if [ -n "$INCLUDED_CONTAINERS" ]; then
  while IFS="$(printf '\t')" read -r CID CNAME; do
    [ -z "$CID" ] && continue
    ENV_GROUP_COUNT=$((ENV_GROUP_COUNT + 1))
    CONTAINER_ENV=$(docker exec "$CID" env 2>/dev/null) || \
      CONTAINER_ENV=$(docker inspect --format '{{`{{range .Config.Env}}{{println .}}{{end}}`}}' "$CID" 2>/dev/null) || true
    if [ -n "$CONTAINER_ENV" ]; then
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
      printf '%s' "Error: could not collect container environment variables" > "${_tmpdir}/envgroup_${ENV_GROUP_COUNT}.txt"
    fi
    printf '%s' "$CNAME" > "${_tmpdir}/envgroup_${ENV_GROUP_COUNT}.name"
    printf '%s' "${CNAME}-container-env" > "${_tmpdir}/envgroup_${ENV_GROUP_COUNT}.resource"
  done <<EOF_INCLUDED
$INCLUDED_CONTAINERS
EOF_INCLUDED
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
  ENV_EXCLUDE_SET=$(parse_exclude_input "$ENV_EXCLUDE_INPUT" "$ENV_GROUP_COUNT")

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

# Collect and upload container logs
if [ -n "$INCLUDED_CONTAINERS" ]; then
  echo ""
  echo "Collecting and uploading container logs..."
  while IFS="$(printf '\t')" read -r CID CNAME; do
    [ -z "$CID" ] && continue
    CONTAINER_LOGS=$(docker logs --tail 1000 "$CID" 2>&1 || true)
    if [ -n "$CONTAINER_LOGS" ]; then
      upload_resource "${CNAME}-container-logs" "$CONTAINER_LOGS"
      echo "  Uploaded logs for $CNAME (last 1000 lines)"
    else
      echo "  No logs available for $CNAME"
    fi
  done <<EOF_INCLUDED
$INCLUDED_CONTAINERS
EOF_INCLUDED
fi

# Finalize support bundle
echo ""
echo "Finalizing support bundle..."
if ! curl -fsSL -X POST "${BASE_URL}/finalize?bundleSecret=${BUNDLE_SECRET}" > /dev/null 2>&1; then
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
