#!/bin/sh
# Builds a Distr Community Edition image from this checkout with nothing but Docker, for testing code that no
# published image contains yet. Onboarding runs the published image instead.
#
#   sh hack/build-ce-image.sh [tag]    tags the image distr-ce:local unless a tag is given
#   DISTR_IMAGE=distr-ce:local sh deploy/docker/quickstart/try.sh
set -eu

cd "$(dirname "$0")/.."
tag="${1:-distr-ce:local}"
# Distr names the controller images it hands out after its own version, so the image takes the version of the last
# release, whose controller images exist, instead of "snapshot".
version="$(sed -n 's/^  "version": "\(.*\)",$/\1/p' package.json)"
commit="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
docker build -f Dockerfile.distr --target local \
  --build-arg VERSION="$version" --build-arg COMMIT="$commit" -t "$tag" .
