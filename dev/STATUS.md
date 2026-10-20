# Status

Last updated: 2026-10-19

## Current focus

Post-MVP — three-tab extension parity shipped; Homebrew spikes still open.

## Active task

_None — next candidate: `spike-homebrew-tap-formula` / `spike-extension-coinstall`, or `upgrade-extensions-typescript-7` (after parity)._

## Blockers

Parent `homebrew-tap-distribution` blocked_by both spikes (OQs open).

## Next action

Run Homebrew spikes, or cut v0.1.0 / commit pending work as user directs.

## Session notes

- Added `.github/workflows/extension-build.yml` — `npm ci` + `npm run build` for Chrome/Firefox on PRs and pushes to `main` (path-filtered). Mark that check required on `main` in GitHub if merges should be blocked.
- Crawl/progress fixes and three-tab parity shipped earlier; Homebrew spikes / v0.1.0 / TS7 upgrade still open.
