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

- Hosted session + single-issue refresh OK. Project crawl failed: stored URL `/browse/SK` was not parsed (only `/browse/KEY-N`); ParseProjectRef now accepts project-only browse URLs; crawl job failures are logged.
- Reload extension from `dist/` after rebuild; restart `make serve` for crawl fix.
- Shipped `extension-three-tab-parity`; Homebrew spikes / v0.1.0 still open.
