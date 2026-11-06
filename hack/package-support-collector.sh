#!/bin/sh
# Packages dist/support-collector for this platform the way the release workflow does, so that
# SUPPORT_COLLECTOR_DOWNLOAD_URL can point the collect script at the local build.
set -eu

release_dir=dist/support-collector-release
archive="support-collector-$(go env GOOS)-$(go env GOARCH).tar.gz"

rm -rf "$release_dir"
mkdir -p "$release_dir"
tar -czf "$release_dir/$archive" -C dist support-collector
cd "$release_dir"
if command -v sha256sum > /dev/null 2>&1; then
  sha256sum "$archive" > support-collector-checksums.txt
else
  shasum -a 256 "$archive" > support-collector-checksums.txt
fi
