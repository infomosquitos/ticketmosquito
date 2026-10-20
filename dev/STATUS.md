# Status

Last updated: 2026-10-19

## Current focus

Post-MVP — TS7 extension toolchain shipped; Homebrew spikes still open.

## Active task

_None — next candidate: `spike-homebrew-tap-formula` / `spike-extension-coinstall`, or cut v0.1.0._

## Blockers

Parent `homebrew-tap-distribution` blocked_by both spikes (OQs open).

## Next action

Run Homebrew spikes, or cut v0.1.0 / commit pending work as user directs.

## Session notes

- Shipped `upgrade-extensions-typescript-7`: both extensions use `tsc --noEmit` + esbuild on TypeScript 7; webpack/ts-loader removed; `engines.node` >=16.20; docs updated.
- `Extension Build` CI already covers Dependabot npm bumps.
- Homebrew spikes / v0.1.0 still open.
