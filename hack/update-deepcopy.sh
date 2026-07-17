#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

DEEPCOPY_GEN="${DEEPCOPY_GEN:-deepcopy-gen}"

"${DEEPCOPY_GEN}" \
  --output-file zz_generated.deepcopy.go \
  github.com/geoberle/pulse/internal/api

# deepcopy-gen emits .DeepCopyInto() calls for time.Time which does not
# implement the interface. Replace with plain value copies.
f="${REPO_ROOT}/internal/api/zz_generated.deepcopy.go"

# Fix pointer-type time.Time fields.
sed -i '' '/\*out = new(time\.Time)/{n;s/(\*in)\.DeepCopyInto(\*out)/**out = **in/;}' "${f}"

# Fix value-type time.Time fields.
for field in CreationTimestamp LastActivity; do
  sed -i '' "s/in\.${field}\.DeepCopyInto(&out\.${field})/out.${field} = in.${field}/g" "${f}"
done
