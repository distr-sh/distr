#!/bin/bash
# Copies every tag of the former agent images, including the cosign signature tags, to the controller
# repositories. Run once, with write access to ghcr.io/distr-sh, before releasing the Distr version whose
# manifests reference the controller images, since targets on older versions are served those names too.
set -euo pipefail

for kind in docker kubernetes; do
  mise exec crane@0.21.7 -- crane copy --all-tags \
    "ghcr.io/distr-sh/distr/$kind-agent" \
    "ghcr.io/distr-sh/distr/$kind-controller"
done
