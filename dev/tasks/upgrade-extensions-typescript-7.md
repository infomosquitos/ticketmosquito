# Upgrade Chrome/Firefox extensions to TypeScript 7

- **Task ID:** `upgrade-extensions-typescript-7`
- **Status:** done
- **Parent / blocked by:** Prefer after `extension-three-tab-parity` (paste SM on TS5 first; see that task OQ9)

## Problem

Dependabot already proposed `typescript@7` for both extensions. Merging it broke `webpack` + `ts-loader` builds: TypeScript 7’s default Node entry no longer exports the classic compiler API (`typescript.sys`, `createProgram`, …) that `ts-loader` expects (`Cannot read properties of undefined (reading 'fileExists')`).

We pinned back to `typescript@~5.9.3` so releases build. Without a deliberate upgrade path, Dependabot will keep proposing TS7 majors and CI/`make build-extensions` will break again if someone merges blindly.

## Goal

Both `chrome-extension` and `firefox-extension` build cleanly on **TypeScript 7** (same toolchain for both), with Dependabot able to track TS7+ without breaking the extension pipeline.

## Open questions

| # | Topic | Status | Decision |
|---|--------|--------|----------|
| 1 | Build approach | locked | **B** — Drop webpack/`ts-loader`. **esbuild** bundles (strip types); **TypeScript 7** (`tsc`/`tsgo --noEmit`) typechecks only. Rejected: **A** (embeds broken compiler API); **C** (tsgo emit still needs a bundler → extra step for the same result). |
| 2 | Typecheck vs emit | locked | Separate typecheck in `npm run build` / CI (`tsc --noEmit` then esbuild). Transpile-only without check is not allowed. |
| 3 | Shared config | locked | Keep duplicated `package.json` / `tsconfig` / `esbuild.mjs` per extension. No shared build package — duplication was fine; esbuild doesn’t change that (separate manifests + Dependabot dirs). |
| 4 | Dependabot guard | locked | **Allow** `typescript` majors (no Dependabot ignore). Guard is green CI (`Extension Build`) + Dependabot’s green-build requirement — not an ignore/pin policy after this task ships. |
| 5 | Minimum Node | locked | Align with TS7: `engines.node` `>=16.20.0`. CI stays on Node **22**. |

## Non-goals

- Migrating the Go backend or any non-extension TypeScript
- Rewriting extension product features
- Adopting TS7 for the whole monorepo beyond the two extension packages

## Implementation

1. Confirm current failure mode (document one-liner in Notes if still accurate).
2. Lock OQ1 after a short spike/prototype on one extension.
3. Apply the same toolchain to Chrome and Firefox; `npm run build` green for both.
4. Update `scripts/build-extension-zips.sh` / CI if scripts change.
5. No Dependabot `ignore` for typescript majors (OQ4 locked allow); rely on `Extension Build` + Dependabot green-build requirement.
6. Note in `docs/INSTALL-FROM-SOURCE.md` or `dev/README.md` if the build prerequisite story changes.

## Done when

- [x] `chrome-extension` and `firefox-extension` `npm ci && npm run build` succeed with `typescript@7` (or current 7.x).
- [x] Typecheck still catches real errors (not transpile-only without a check step, unless OQ2 explicitly allows).
- [x] Dependabot allowed to propose typescript majors; broken bumps fail `Extension Build` (green-build required).
- [x] Brief note in Notes or README on how to build extensions after the change.

## Notes

### Why TS7 broke us (2026-10)

```text
require('typescript')  // TS 5.x → full compiler API including ts.sys.fileExists
require('typescript')  // TS 7.0 → { version, versionMajorMinor } only
```

TS7 is the native/`tsgo`-oriented package surface; `ts-loader@9` still assumes the TS5 JS API. PeerDependency `typescript: '*'` does **not** mean runtime compatibility.

### Shipped approach (OQ1 B)

```text
npm run build  →  tsc --noEmit  &&  node esbuild.mjs
```

- Deps: `typescript@^7`, `esbuild`, `@types/chrome` (removed `webpack`, `webpack-cli`, `ts-loader`, `copy-webpack-plugin`).
- esbuild: three browser entries (`background`, `popup`, `content`) + copy manifest/HTML/CSS/assets.
- Same toolchain for Chrome and Firefox; `engines.node`: `>=16.20.0`.

### CI

`Extension Build` (`.github/workflows/extension-build.yml`) runs `npm ci && npm run build` for both extensions on PRs / pushes when extension paths change.
