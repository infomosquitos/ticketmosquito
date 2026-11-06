# Status

Last updated: 2026-11-06

## Current focus

Post-MVP — Firefox AMO signing wired; cutting **v0.2.1** to publish the first `.xpi`.

## Active task

Firefox XPI signing in release CI (in progress → verify on v0.2.1).

## Blockers

_None for signed Firefox XPI._ `spike-extension-coinstall` (assisted CLI / share layout) remains open product work, not blocked on Mozilla keys.

## Next action

Confirm v0.2.1 Release includes `ticketmosquito-firefox-v0.2.1.xpi`, then resume co-install spike design if desired.

## Session notes

- Ported SM `web-ext sign` into `scripts/build-extension-zips.sh`; release job sets `REQUIRE_FIREFOX_XPI=1` + AMO secrets.
- AMO secrets were already on the repo; gecko id `ticket@mosqui.to` already in the Firefox manifest.
