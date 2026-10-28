{{template "collector-start" .}}

# The collector is released with Distr. Download the build for this platform and check it against
# the checksums of the same release before running it.
COLLECTOR_DOWNLOAD_URL="{{.CollectorDownloadURL}}"

collector_platform() {
  case "$(uname -s)" in
    Linux) _os=linux ;;
    Darwin) _os=darwin ;;
    *)
      echo "Error: the Kubernetes collector is not available for $(uname -s)." >&2
      return 1
      ;;
  esac
  case "$(uname -m)" in
    x86_64|amd64) _arch=amd64 ;;
    aarch64|arm64) _arch=arm64 ;;
    *)
      echo "Error: the Kubernetes collector is not available for $(uname -m)." >&2
      return 1
      ;;
  esac
  printf '%s-%s' "$_os" "$_arch"
}

sha256_of() {
  if command -v sha256sum > /dev/null 2>&1; then
    sha256sum "$1" | cut -d ' ' -f 1
  elif command -v shasum > /dev/null 2>&1; then
    shasum -a 256 "$1" | cut -d ' ' -f 1
  else
    return 1
  fi
}

if [ -n "${DISTR_SUPPORT_BUNDLE_COLLECTOR:-}" ]; then
  _collector="$DISTR_SUPPORT_BUNDLE_COLLECTOR"
  echo "Using the collector at ${_collector}"
else
  _platform=$(collector_platform) || exit 1
  _archive="kubernetes-support-bundle-${_platform}.tar.gz"
  echo "Downloading the Kubernetes collector for ${_platform}..."
  if ! curl -fsSL -o "${_tmpdir}/${_archive}" "${COLLECTOR_DOWNLOAD_URL}/${_archive}" ||
    ! curl -fsSL -o "${_tmpdir}/checksums.txt" "${COLLECTOR_DOWNLOAD_URL}/kubernetes-support-bundle-checksums.txt"; then
    echo "Error: the collector could not be downloaded from ${COLLECTOR_DOWNLOAD_URL}." >&2
    echo "Allow this machine to reach it, then run the command again." >&2
    exit 1
  fi
  _expected=$(awk -v name="$_archive" '$2 == name { print $1 }' "${_tmpdir}/checksums.txt")
  if ! _actual=$(sha256_of "${_tmpdir}/${_archive}"); then
    echo "Error: neither sha256sum nor shasum is available to verify the collector." >&2
    exit 1
  elif [ -z "$_expected" ] || [ "$_expected" != "$_actual" ]; then
    echo "Error: the downloaded collector does not match the checksum of its release." >&2
    exit 1
  fi
  mkdir "${_tmpdir}/collector"
  if ! tar -xzf "${_tmpdir}/${_archive}" -C "${_tmpdir}/collector"; then
    echo "Error: the downloaded collector could not be unpacked." >&2
    exit 1
  fi
  _collector="${_tmpdir}/collector/kubernetes-support-bundle"
fi

# The collector writes the bundle into the temp dir, which is removed on exit, and asks before it
# uploads anything. Arguments passed to this script (sh -s -- --context prod) go to the collector.
echo ""
DISTR_SUPPORT_BUNDLE_URL="$BASE_URL" DISTR_SUPPORT_BUNDLE_SECRET="$BUNDLE_SECRET" \
  "$_collector" --output "${_tmpdir}/bundle" \
  --log-tail-lines {{.LogTail}} --max-resource-bytes {{.ResourceMaxBytes}} "$@"
_code=$?
if [ "$_code" -eq 126 ]; then
  echo "Error: the collector cannot be executed from ${_tmpdir}." >&2
  echo "Set TMPDIR to a directory that allows running programs, then run the command again." >&2
  exit 1
elif [ "$_code" -ne 0 ]; then
  echo "The support bundle was left open, so the same command can be run again once the problem above is fixed." >&2
  exit 1
fi
{{- template "collector-end" .}}
