#!/usr/bin/env bash
# Rasterize toolbar PNGs from assets/icon.svg (active) and a gray inactive variant.
# Prefer rsvg-convert (librsvg). Usage: ./scripts/render-extension-icons.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# Muted blue brand stroke (was SpaceMosquito green #4a7a5c)
ACTIVE_COLOR="#4a6f8c"
INACTIVE_COLOR="#8a93a0"
SIZES=(16 32 48 128)

render_size() {
  local src="$1" size="$2" out="$3"
  if command -v rsvg-convert >/dev/null 2>&1; then
    rsvg-convert -w "$size" -h "$size" -b none "$src" -o "$out"
  elif command -v magick >/dev/null 2>&1; then
    echo "warning: rsvg-convert not found; ImageMagick may crop poorly" >&2
    magick -background none "$src" \
      -trim +repage \
      -resize "${size}x${size}" \
      -gravity center -background none -extent "${size}x${size}" \
      -depth 8 "PNG32:$out"
  else
    return 2
  fi
}

if ! command -v rsvg-convert >/dev/null 2>&1 && ! command -v magick >/dev/null 2>&1; then
  echo "need rsvg-convert (brew install librsvg) or magick to render icons" >&2
  exit 1
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

for dir in chrome-extension firefox-extension; do
  dest="$ROOT/$dir/assets"
  active="$dest/icon.svg"
  gray="$TMP/icon-inactive.svg"
  sed "s/${ACTIVE_COLOR}/${INACTIVE_COLOR}/g" "$active" > "$gray"
  for s in "${SIZES[@]}"; do
    render_size "$active" "$s" "$dest/icon-active-${s}.png"
    render_size "$gray" "$s" "$dest/icon-inactive-${s}.png"
  done
  echo "rendered $dir/assets/icon-{active,inactive}-{16,32,48,128}.png"
done
