#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"

make deps
make validate
make test
make test-race
make test-integration
make build-binaries
make checksums
make verify-artifacts

echo "local CI checks passed"
