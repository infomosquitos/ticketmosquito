# Status

Last updated: 2026-11-06

## Current focus

Post-MVP — Homebrew cask shipped in **v0.2.0**; co-install spike still blocked on Mozilla keys.

## Active task

_None — next candidate: `spike-extension-coinstall` (blocked on Mozilla AMO/signing keys)._

## Blockers

- **Mozilla AMO/signing keys** — durable Firefox XPI / finishing `spike-extension-coinstall`. Cask already ships with temporary Firefox load caveats.

## Next action

When Mozilla keys are available: finish `spike-extension-coinstall` (parent `homebrew-tap-distribution` still open).

## Session notes

- Shipped `homebrew-tap-cask`: render script, release `homebrew` job, tap README row.
- [v0.2.0](https://github.com/infomosquitos/ticketmosquito/releases/tag/v0.2.0) published (binaries + chrome/firefox zips); tap cask + `brew install --cask infomosquitos/tap/ticketmosquito` smoke OK (`ticketmosquito version` → `v0.2.0`).
