# Upgrade Chrome/Firefox extensions to TypeScript 7

- **Task ID:** `upgrade-extensions-typescript-7`
- **Status:** backlog
- **Parent / blocked by:** Prefer after `extension-three-tab-parity` (paste SM on TS5 first; see that task OQ9)

## Problem

Dependabot already proposed `typescript@7` for both extensions. Merging it broke `webpack` + `ts-loader` builds: TypeScript 7’s default Node entry no longer exports the classic compiler API (`typescript.sys`, `createProgram`, …) that `ts-loader` expects (`Cannot read properties of undefined (reading 'fileExists')`).

We pinned back to `typescript@~5.9.3` so releases build. Without a deliberate upgrade path, Dependabot will keep proposing TS7 majors and CI/`make build-extensions` will break again if someone merges blindly.

## Goal

Both `chrome-extension` and `firefox-extension` build cleanly on **TypeScript 7** (same toolchain for both), with Dependabot able to track TS7+ without breaking the extension pipeline.

## Open questions

| # | Topic | Status | Decision |
|---|--------|--------|----------|
| 1 | Build approach | open | **A)** Keep webpack; replace/adapt loader once TS7-compatible (`ts-loader` successor, custom loader, or `tsgo` emit + copy). **B)** Switch extensions to `esbuild`/`tsx` (transpile-only) + keep `tsc`/TS7 for typecheck. **C)** Native `tsgo`/TS7 CLI as primary emit, webpack only for bundling JS. |
| 2 | Typecheck vs emit | open | Strict: `tsc`/`tsgo --noEmit` in CI separate from bundler; or single-step typed emit if tooling supports it |
| 3 | Shared config | open | Keep duplicated `package.json`/`tsconfig` per extension vs extract shared build config package |
| 4 | Dependabot guard | open | After upgrade: allow `typescript` majors; until then ignore `typescript` major in both npm groups (or pin `~5.9`) |
| 5 | Minimum Node | open | Align with TS7 `engines` (`>=16.20`) and current CI Node 22 — confirm no lower bound needed locally |

## Non-goals

- Migrating the Go backend or any non-extension TypeScript
- Rewriting extension product features
- Adopting TS7 for the whole monorepo beyond the two extension packages

## Implementation

1. Confirm current failure mode (document one-liner in Notes if still accurate).
2. Lock OQ1 after a short spike/prototype on one extension.
3. Apply the same toolchain to Chrome and Firefox; `npm run build` green for both.
4. Update `scripts/build-extension-zips.sh` / CI if scripts change.
5. Adjust `.github/dependabot.yml` ignore rules once TS7 is green (or add ignores until then).
6. Note in `docs/INSTALL-FROM-SOURCE.md` or `dev/README.md` if the build prerequisite story changes.

## Done when

- [ ] `chrome-extension` and `firefox-extension` `npm ci && npm run build` succeed with `typescript@7` (or current 7.x).
- [ ] Typecheck still catches real errors (not transpile-only without a check step, unless OQ2 explicitly allows).
- [ ] Dependabot policy updated so TS7 isn’t a footgun (allow or ignore majors intentionally).
- [ ] Brief note in Notes or README on how to build extensions after the change.

## Notes

### Why TS7 broke us (2026-10)

```text
require('typescript')  // TS 5.x → full compiler API including ts.sys.fileExists
require('typescript')  // TS 7.0 → { version, versionMajorMinor } only
```

TS7 is the native/`tsgo`-oriented package surface; `ts-loader@9` still assumes the TS5 JS API. PeerDependency `typescript: '*'` does **not** mean runtime compatibility.

### Suggested leaning (non-locked)

Prefer **OQ1 B or C**: stop relying on `ts-loader` for typechecked emit; use a modern bundler transpile + explicit `tsgo`/`tsc` typecheck in `npm run build` (or a `prebuild` script). Keeps webpack if we want, without waiting on `ts-loader`.

### Pin today

```json
"typescript": "~5.9.3"
```

in both extension `package.json` files until this task ships.

### CI

`Extension Build` (`.github/workflows/extension-build.yml`) runs `npm ci && npm run build` for both extensions on PRs / pushes to `main` when extension paths change — enough to fail a Dependabot TS7 bump before merge (if the check is required on the branch).
