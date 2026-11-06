# Status

Last updated: 2026-11-06

## Current focus

Cutting **v0.2.0** with Homebrew cask automation (`homebrew-tap-cask`).

## Active task

`homebrew-tap-cask` — in progress (script + release job landed; awaiting tag/CI/smoke).

## Blockers

- **Mozilla AMO/signing keys** — only for durable Firefox XPI / finishing `spike-extension-coinstall`. Does **not** block the cask.

## Next action

Push `v0.2.0`, confirm Release + tap update, smoke `brew install --cask infomosquitos/tap/ticketmosquito`.

## Session notes

- `scripts/render-homebrew-cask.sh` + `homebrew` job in `release.yml`.
- Docs point at brew as primary macOS install; Firefox caveat remains temporary/unsigned zip.
