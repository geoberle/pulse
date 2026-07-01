#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

DEEPCOPY_GEN="${DEEPCOPY_GEN:-deepcopy-gen}"

"${DEEPCOPY_GEN}" \
  --output-file zz_generated.deepcopy.go \
  --go-header-file "${REPO_ROOT}/hack/boilerplate.go.txt" \
  github.com/geoberle/pulse/internal/workitem
