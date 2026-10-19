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

- Hosted Jira validate 401 again with only 3 cookies after name-dedupe; capture now uses identity dedupe + CHIPS-safe queries (`domain` + `partitionKey:{}`, not `url`+partition) and logs `capture_debug` on POST /api/session.
- Reload extension from `dist/` and recapture; expect cookie_count > 3 and/or capture_debug.queryHits showing partAny hits.
- Shipped `extension-three-tab-parity`; Homebrew spikes / v0.1.0 still open.
