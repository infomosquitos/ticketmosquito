# Status

Last updated: 2026-11-06

## Current focus

Post-MVP — signed Firefox XPI shipping; co-install spike is the remaining distribution UX work.

## Active task

_None — next candidate: `spike-extension-coinstall` (ready)._

## Blockers

_None._

## Next action

Design/lock co-install spike OQs (CLI + on-disk layout); parent `homebrew-tap-distribution` waits on that.

## Session notes

- [v0.2.1](https://github.com/infomosquitos/ticketmosquito/releases/tag/v0.2.1): AMO-signed `ticketmosquito-firefox-v0.2.1.xpi` + chrome zip; tap caveats updated.
- Mozilla keys were already present; only the SM-style `web-ext sign` path was missing.
