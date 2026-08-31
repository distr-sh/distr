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

upload_resource_file() {
  _name="$1"
  _file="$2"
  _errfile="${_tmpdir}/upload_err.tmp"
  if ! curl -fsSL -X POST \
    -F "name=${_name}" \
    -F "content=@${_file}" \
    "${BASE_URL}/resources?bundleSecret=${BUNDLE_SECRET}" > /dev/null 2>"$_errfile"; then
    _err=$(cat "$_errfile" 2>/dev/null)
    if [ -n "$_err" ]; then
      echo "    Warning: failed to upload ${_name}: ${_err}"
    else
      echo "    Warning: failed to upload ${_name}"
    fi
  fi
  rm -f "$_errfile"
}

upload_resource() {
  _tmpfile="${_tmpdir}/upload_content.tmp"
  printf '%s' "$2" > "$_tmpfile"
  upload_resource_file "$1" "$_tmpfile"
  rm -f "$_tmpfile"
}
{{if .Scripts}}
# Run a custom script with the interpreter named in its shebang, falling back to sh. The
# interpreter is deliberately unquoted so that "#!/usr/bin/env bash" splits into command and
# argument. Executing the file directly instead would fail when the temp dir is mounted noexec.
run_custom_script() {
  _interp=$(sed -n '1s/^#![[:space:]]*//p' "$1" | tr -d '\r')
  if [ -n "$_interp" ] && command -v "${_interp%% *}" > /dev/null 2>&1; then
    $_interp "$1"
  else
    sh "$1"
  fi
}
{{end}}
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

{{- if .Scripts}}

# Run the custom scripts configured by the vendor and collect their output. Their names,
# descriptions and bodies are embedded base64-encoded so that nothing in them can terminate the
# heredoc above, which would make the rest of this file run as the outer script.
echo ""
echo "Preparing custom scripts..."
SCRIPT_COUNT=0
if ! command -v base64 > /dev/null 2>&1; then
  echo "  Warning: base64 is not available, skipping custom scripts"
else
  mkdir -p "${_tmpdir}/scripts"
{{- range .Scripts}}
  SCRIPT_COUNT=$((SCRIPT_COUNT + 1))
  printf '%s' '{{.NameBase64}}' | base64 -d > "${_tmpdir}/scripts/${SCRIPT_COUNT}.name"
  printf '%s' '{{.DescriptionBase64}}' | base64 -d > "${_tmpdir}/scripts/${SCRIPT_COUNT}.desc"
  printf '%s' '{{.ContentBase64}}' | base64 -d > "${_tmpdir}/scripts/${SCRIPT_COUNT}.sh"
{{- end}}
fi

if [ "$SCRIPT_COUNT" -gt 0 ]; then
  echo ""
  echo "Your vendor provided the following scripts to run on this host:"
  echo "---"
  _s=1
  while [ "$_s" -le "$SCRIPT_COUNT" ]; do
    printf "  [%d] %s\n" "$_s" "$(cat "${_tmpdir}/scripts/${_s}.name")"
    _sdesc=$(cat "${_tmpdir}/scripts/${_s}.desc")
    if [ -n "$_sdesc" ]; then
      printf "      %s\n" "$_sdesc"
    fi
    echo ""
    while IFS= read -r _line || [ -n "$_line" ]; do
      printf "      %s\n" "$_line"
    done < "${_tmpdir}/scripts/${_s}.sh"
    echo ""
    _s=$((_s + 1))
  done

  echo "Enter script numbers to EXCLUDE from running (comma-separated), or press Enter to run all:"
  read -r SCRIPT_EXCLUDE_INPUT
  SCRIPT_EXCLUDE_SET=$(parse_exclude_input "$SCRIPT_EXCLUDE_INPUT" "$SCRIPT_COUNT")

  echo ""
  SCRIPT_OUTPUT_COUNT=0
  _s=1
  while [ "$_s" -le "$SCRIPT_COUNT" ]; do
    _sname=$(cat "${_tmpdir}/scripts/${_s}.name")
    _sbase="${_tmpdir}/scripts/${_s}"
    if [ -n "$SCRIPT_EXCLUDE_SET" ] && echo "$SCRIPT_EXCLUDE_SET" | grep -q ",$_s,"; then
      echo "  Skipping ${_sname}"
    else
      echo "  Running ${_sname}..."
      # Custom scripts must not consume the stdin the collector reads its prompts from.
      run_custom_script "${_sbase}.sh" > "${_sbase}.raw" 2> "${_sbase}.err" < /dev/null
      _scode=$?
      head -c {{.ResourceMaxBytes}} "${_sbase}.raw" > "${_sbase}.out"
      # stdout and stderr share the per-resource budget, so one script cannot contribute twice the cap.
      _sbudget=$(({{.ResourceMaxBytes}} - $(wc -c < "${_sbase}.out")))
      if [ "$(wc -c < "${_sbase}.raw")" -gt {{.ResourceMaxBytes}} ]; then
        printf '\n--- distr: output truncated at %s bytes ---\n' "{{.ResourceMaxBytes}}" >> "${_sbase}.out"
      fi
      if [ "$_scode" -ne 0 ]; then
        printf '\n--- distr: script exited with code %s ---\n' "$_scode" >> "${_sbase}.out"
      fi
      if [ -s "${_sbase}.err" ]; then
        printf '\n--- distr: stderr ---\n' >> "${_sbase}.out"
        if [ "$_sbudget" -gt 0 ]; then
          head -c "$_sbudget" "${_sbase}.err" >> "${_sbase}.out"
        fi
        if [ "$(wc -c < "${_sbase}.err")" -gt "$_sbudget" ]; then
          printf '\n--- distr: stderr truncated at %s bytes of total output ---\n' \
            "{{.ResourceMaxBytes}}" >> "${_sbase}.out"
        fi
      fi
      SCRIPT_OUTPUT_COUNT=$((SCRIPT_OUTPUT_COUNT + 1))
    fi
    _s=$((_s + 1))
  done

  if [ "$SCRIPT_OUTPUT_COUNT" -gt 0 ]; then
    echo ""
    echo "Script output to upload:"
    echo "---"
    _s=1
    while [ "$_s" -le "$SCRIPT_COUNT" ]; do
      _sbase="${_tmpdir}/scripts/${_s}"
      if [ -f "${_sbase}.out" ]; then
        printf "  %s\n" "$(cat "${_sbase}.name")"
        _slines=$(wc -l < "${_sbase}.out")
        head -n 30 "${_sbase}.out" | while IFS= read -r _line || [ -n "$_line" ]; do
          printf "      %s\n" "$_line"
        done
        if [ "$_slines" -gt 30 ]; then
          printf "      ... (%s more lines)\n" "$((_slines - 30))"
        fi
        echo ""
      fi
      _s=$((_s + 1))
    done

    printf "Upload script output? [Y/n]: "
    read -r SCRIPT_CONFIRM
    case "$SCRIPT_CONFIRM" in
      [nN]*)
        echo "  Skipping script output upload"
        ;;
      *)
        _s=1
        while [ "$_s" -le "$SCRIPT_COUNT" ]; do
          _sbase="${_tmpdir}/scripts/${_s}"
          if [ -f "${_sbase}.out" ]; then
            _sname=$(cat "${_sbase}.name")
            upload_resource_file "$_sname" "${_sbase}.out"
            echo "  Uploaded output of ${_sname}"
          fi
          _s=$((_s + 1))
        done
        ;;
    esac
  fi
fi
{{- end}}

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
