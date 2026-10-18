#!/usr/bin/env bash
# Build Chrome (and unsigned Firefox) extension zips for releases.
# Usage: build-extension-zips.sh <version> [out_dir]
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:?usage: build-extension-zips.sh <version> [out_dir]}"
OUT_DIR="${2:-$REPO_ROOT/dist}"
MANIFEST_VERSION="${VERSION#v}"

mkdir -p "$OUT_DIR"
OUT_DIR="$(cd "$OUT_DIR" && pwd)"

stamp_manifest() {
  local path="$1"
  local ver="$2"
  node -e '
    const fs = require("fs");
    const path = process.argv[1];
    const ver = process.argv[2];
    const m = JSON.parse(fs.readFileSync(path, "utf8"));
    m.version = ver;
    fs.writeFileSync(path, JSON.stringify(m, null, 2) + "\n");
  ' "$path" "$ver"
}

build_one() {
  local name="$1"
  local dir="$REPO_ROOT/${name}-extension"
  local zip_path="$OUT_DIR/ticketmosquito-${name}-${VERSION}.zip"
  local bak="${dir}/manifest.json.bak.$$"

  echo "Building ${name}-extension (manifest $MANIFEST_VERSION)"
  cp "${dir}/manifest.json" "$bak"
  stamp_manifest "${dir}/manifest.json" "$MANIFEST_VERSION"
  (
    cd "$dir"
    if [[ -f package-lock.json ]]; then
      npm ci
    else
      npm install
    fi
    npm run build
  )
  rm -f "$zip_path"
  ( cd "$dir/dist" && zip -r "$zip_path" . )
  mv "$bak" "${dir}/manifest.json"
  echo "Wrote $zip_path"
}

build_one chrome
build_one firefox

echo "Extension packages in $OUT_DIR"
