{{template "collector-start" .}}

# Docker is the only part of this script that needs elevated privileges: the
# daemon socket is usually owned by root:docker. If the docker CLI is installed
# but we cannot reach the daemon because of a permission error (and we are not
# already root), collecting container data would be silently skipped. Exit early
# with a copy-pasteable command to re-run the collector with sudo instead.
if command -v docker >/dev/null 2>&1 && [ "$(id -u 2>/dev/null)" != "0" ]; then
  DOCKER_ERR=$(docker ps 2>&1 >/dev/null)
  if echo "$DOCKER_ERR" | grep -qi "permission denied"; then
    echo "Error: accessing the Docker daemon requires elevated privileges." >&2
    echo "Please re-run the collector with sudo:" >&2
    echo "" >&2
    echo "  curl -fsSL '${BASE_URL}/collect-script?bundleSecret=${BUNDLE_SECRET}' | sudo sh" >&2
    echo "" >&2
    exit 1
  fi
fi

# Probe the Docker daemon once, up front. Whether the daemon is reachable is
# critical diagnostic information (it may be installed but not running), so the
# status is always included with the system information below. The container
# list is captured here and reused for the log/env collection further down.
DOCKER_STATUS="Docker CLI not found in PATH"
CONTAINERS=""
if command -v docker >/dev/null 2>&1; then
  if CONTAINERS=$(docker ps -a --format "{{`{{.ID}}`}}	{{`{{.Names}}`}}	{{`{{.Status}}`}}	{{`{{.Image}}`}}" 2>/dev/null); then
    DOCKER_STATUS="daemon reachable
$(docker version 2>&1 || true)"
  else
    CONTAINERS=""
    DOCKER_STATUS="daemon unavailable
$(docker version 2>&1 || true)"
  fi
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
$(free -h 2>/dev/null || echo 'unavailable')

docker:
$DOCKER_STATUS"

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

# Detect Docker containers and build included container list (reusing the list
# captured during the Docker probe above)
echo ""
echo "Detecting Docker containers..."

CONTAINER_COUNT=0
INCLUDED_CONTAINERS=""
if [ -z "$CONTAINERS" ]; then
  echo "  No Docker containers found (see Docker status in system information)"
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
    while IFS= read -r _line || [ -n "$_line" ]; do
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
  _lograw="${_tmpdir}/container_logs.raw"
  _logout="${_tmpdir}/container_logs.out"
  while IFS="$(printf '\t')" read -r CID CNAME; do
    [ -z "$CID" ] && continue
    docker logs --tail {{.LogTail}} "$CID" > "$_lograw" 2>&1 || true
    if [ -s "$_lograw" ]; then
      # Logs are cut off at the front, unlike script output: the most recent lines matter most.
      if [ "$(wc -c < "$_lograw")" -gt {{.ResourceMaxBytes}} ]; then
        printf '%s\n' "--- distr: earlier output truncated at {{.ResourceMaxBytes}} bytes ---" > "$_logout"
        tail -c {{.ResourceMaxBytes}} "$_lograw" >> "$_logout"
        _lognote=" (truncated to {{.ResourceMaxBytes}} bytes)"
      else
        cat "$_lograw" > "$_logout"
        _lognote=""
      fi
      upload_resource_file "${CNAME}-container-logs" "$_logout"
      echo "  Uploaded logs for $CNAME${_lognote}"
    else
      echo "  No logs available for $CNAME"
    fi
    rm -f "$_lograw" "$_logout"
  done <<EOF_INCLUDED
$INCLUDED_CONTAINERS
EOF_INCLUDED
fi
{{- template "collector-end" .}}
