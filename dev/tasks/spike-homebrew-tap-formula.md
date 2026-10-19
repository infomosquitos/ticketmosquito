# Spike: Homebrew tap + formula/cask shape

- **Task ID:** `spike-homebrew-tap-formula`
- **Status:** backlog
- **Parent / blocked by:** parent: `homebrew-tap-distribution`

## Problem

TicketMosquito can already ship GitHub Release binaries, but there is no opinionated macOS (and optional Linuxbrew) install path. Formula vs cask, bottle source, and multi-arch naming must be decided before a tap is useful.

## Goal

A short written recommendation: tap repo layout, formula and/or cask, how artifacts map from current `release.yml` assets, and what is **out of brew’s reach** (browser extensions — defer to sibling spike).

## Open questions

| # | Topic | Status | Decision |
|---|--------|--------|----------|
| 1 | Tap hosting | open | Own GitHub org/user tap (e.g. `infomosquitos/homebrew-ticketmosquito`) vs formula in a shared mosquito tap? |
| 2 | Formula vs cask | open | **Likely formula** for CLI binary on PATH; cask only if we ship a `.app` / GUI wrapper later |
| 3 | Artifact source | open | Prefer bottles from GitHub Releases (`ticketmosquito-darwin-{arm64,amd64}`) + `sha256` from `SHA256SUMS`; live `go build` as fallback? |
| 4 | Linux | open | Support Linuxbrew with `ticketmosquito-linux-amd64` or macOS-only for v1? |
| 5 | Version coupling | open | Formula version == git tag `vX.Y.Z`; document bump/update flow (manual PR to tap vs automation) |

## Non-goals

- Implementing the tap
- Chrome/Firefox store publishing
- Windows package managers (Scoop/Winget) — note only

## Implementation

Investigate current CI artifacts (`.github/workflows/release.yml`, `scripts/build-release.sh`). Skim Homebrew docs for `url`/`sha256`/`bottle` patterns for Go single-binaries. Compare SpaceMosquito (none today) and similar CLI+extension products if useful.

## Done when

- [ ] Recommendation table: formula vs cask, platforms, bottle URL pattern, tap repo name suggestion.
- [ ] Explicit “extensions are not installed by brew alone” + pointer to `spike-extension-coinstall`.
- [ ] Open questions locked or narrowed to 0–1 remaining for the parent task.

## Notes

### What brew can distribute cleanly today

| Artifact | Brew-friendly? | Notes |
|----------|----------------|-------|
| `ticketmosquito` binary | Yes | Formula `bin.install` |
| `LICENSE`, man page, shell completions | Yes | Optional formula extras |
| Chrome zip / Firefox zip | Partial | Can **install files** under `share/ticketmosquito/extensions/`; cannot register with the browser |
| Signed Firefox `.xpi` | Partial | Same; user still confirms install |
| Embedded Chromium / go-rod | N/A for MVP | Not required for API-only crawl |

### Suggested formula sketch (non-normative)

```ruby
class Ticketmosquito < Formula
  desc "Local Jira mirror with search and MCP"
  homepage "https://github.com/infomosquitos/ticketmosquito"
  version "0.1.0"
  # url/sha256 from GitHub Release asset for Hardware::CPU
  # or: depends_on "go" => :build + install from source tarball
end
```

`caveats` should print extension install next steps (exact UX owned by co-install spike).
