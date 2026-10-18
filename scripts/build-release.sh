#!/usr/bin/env bash
# Build multi-arch binaries + extension zips into dist/.
# Usage: build-release.sh <version>
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:?usage: build-release.sh <version>}"
OUT="$REPO_ROOT/dist"
MODULE="github.com/infomosquitos/ticketmosquito/internal/cliapp.Version"

mkdir -p "$OUT"
cd "$REPO_ROOT"

build_bin() {
  local goos="$1" goarch="$2"
  local name="ticketmosquito-${goos}-${goarch}"
  if [[ "$goos" == "windows" ]]; then
    name="${name}.exe"
  fi
  echo "Building $name"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
    -ldflags "-s -w -X ${MODULE}=${VERSION}" \
    -o "$OUT/$name" \
    ./cmd/ticketmosquito
}

build_bin darwin arm64
build_bin darwin amd64
build_bin linux amd64
build_bin windows amd64

chmod +x "$REPO_ROOT/scripts/build-extension-zips.sh"
"$REPO_ROOT/scripts/build-extension-zips.sh" "$VERSION" "$OUT"

(
  cd "$OUT"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum ticketmosquito-* > SHA256SUMS
  else
    shasum -a 256 ticketmosquito-* > SHA256SUMS
  fi
)

echo "Release artifacts in $OUT"
ls -la "$OUT"
