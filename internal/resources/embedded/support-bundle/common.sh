{{- /*
The blocks below are shared by the collect script of every deployment type. Neither emits a
newline at its start or end, so the blank lines around them belong to the template using them.
*/ -}}

{{define "collector-start" -}}
#!/bin/sh

need_tty=yes
_dir=$(mktemp -d)
_script="${_dir}/distr-collect.sh"
# dash and other shells skip the EXIT trap when a signal ends them, so exit on the signals instead.
trap 'rm -rf "$_dir"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

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
trap 'exit 130' INT
trap 'exit 143' TERM

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
# interpreter and the timeout prefix are deliberately unquoted so that "#!/usr/bin/env bash" splits
# into command and argument. Executing the file directly instead would fail when the temp dir is
# mounted noexec.
run_custom_script() {
  _interp=$(sed -n '1s/^#![[:space:]]*//p' "$1" | tr -d '\r')
  if [ -z "$_interp" ] || ! command -v "${_interp%% *}" > /dev/null 2>&1; then
    _interp=sh
  fi
  $SCRIPT_TIMEOUT $_interp "$1"
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
{{- end}}

{{define "collector-end"}}
{{- if .Scripts}}

# Run the custom scripts configured by the vendor and collect their output. Their names,
# descriptions and bodies are embedded base64-encoded so that nothing in them can terminate the
# heredoc above, which would make the rest of this file run as the outer script.
echo ""
echo "Preparing custom scripts..."
SCRIPT_COUNT=0
SCRIPT_TIMEOUT=""
if ! command -v base64 > /dev/null 2>&1; then
  echo "  Warning: base64 is not available, skipping custom scripts"
else
  mkdir -p "${_tmpdir}/scripts"
{{- if .ScriptTimeoutSeconds}}
  # timeout is in coreutils and busybox, but not on every host (macOS ships without it).
  if command -v timeout > /dev/null 2>&1; then
    SCRIPT_TIMEOUT="timeout -k 5 {{.ScriptTimeoutSeconds}}"
  else
    echo "  Warning: timeout is not available, scripts run without a time limit"
  fi
{{- end}}
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
      # coreutils reports a timeout as 124, and as 128+n when the script had to be signalled.
      if [ -n "$SCRIPT_TIMEOUT" ] && { [ "$_scode" -eq 124 ] || [ "$_scode" -eq 137 ] || [ "$_scode" -eq 143 ]; }; then
        printf '\n--- distr: script timed out after %ss ---\n' \
          "{{.ScriptTimeoutSeconds}}" >> "${_sbase}.out"
      elif [ "$_scode" -ne 0 ]; then
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
  sh "$_script" "$@" < /dev/tty
else
  sh "$_script" "$@"
fi
{{- end}}
