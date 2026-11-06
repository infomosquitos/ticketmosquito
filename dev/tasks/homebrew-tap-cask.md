# Ship TicketMosquito Homebrew cask

- **Task ID:** `homebrew-tap-cask`
- **Status:** done
- **Parent / blocked by:** parent: `homebrew-tap-distribution`
- **Renamed from:** `spike-homebrew-tap-formula` (spike complete; this is the implementation task)

## Problem

macOS users lack a first-class install path. Decisions are locked (former spike); next step is to ship the cask on the shared org tap and automate bumps on release.

## Goal

```sh
brew install --cask infomosquitos/tap/ticketmosquito
```

installs a working `ticketmosquito` binary on PATH for macOS arm64/amd64, with caveats for init/serve/extensions (SM-shaped). Release tags update the tap automatically.

## Decisions (locked — former spike OQs)

| # | Topic | Decision |
|---|--------|----------|
| 1 | Tap | Shared [`infomosquitos/homebrew-tap`](https://github.com/infomosquitos/homebrew-tap) — add `Casks/ticketmosquito.rb` beside `spacemosquito` |
| 2 | Shape | **Cask** (not formula): `binary` + quarantine `postflight` + `zap ~/.ticketmosquito` |
| 3 | Artifacts | GitHub Release `ticketmosquito-darwin-{arm64,amd64}` + per-arch `sha256` (same as SM; not Cellar bottles) |
| 4 | Linux | macOS brew only for v1; keep shipping `ticketmosquito-linux-amd64` on Releases + docs |
| 5 | Bumps | Cask `version` == tag `vX.Y.Z` (strip `v`); adapt SM `render-homebrew-cask.sh` + release CI push/PR to the tap |

Extensions are **not** installed by brew. Caveats (and later `spike-extension-coinstall`) cover the browser steps. **Mozilla keys do not block this task** — use SM-style caveats pointing at Release chrome zip / firefox zip (or temporary add-on) until signed XPI exists.

## Non-goals

- Linuxbrew formula
- Chrome Web Store / AMO publishing
- CLI `extension install` helper (owned by co-install spike / follow-up)
- Notarizing darwin binaries (match SM: strip quarantine in postflight)

## Implementation

1. Adapt SpaceMosquito `scripts/render-homebrew-cask.sh` → TicketMosquito (cask name, asset names, caveats, `~/.ticketmosquito`).
2. Wire release workflow to render + push/PR `Casks/ticketmosquito.rb` on `infomosquitos/homebrew-tap`.
3. Update tap README table (one row for TicketMosquito).
4. Point project README / INSTALL at `brew install --cask infomosquitos/tap/ticketmosquito`.
5. Smoke: install from a real GitHub Release tag (needs at least one published `v*` with darwin assets).

## Done when

- [x] `Casks/ticketmosquito.rb` lives on `infomosquitos/homebrew-tap` and installs via brew for arm64/amd64.
- [x] Release automation updates version/sha256 (no hand-edit of the cask).
- [x] Caveats cover init, serve, extension load from Release (honest about Firefox until XPI).
- [x] Docs list brew as a primary install path.

## Notes

### SM pattern to copy

```ruby
cask "ticketmosquito" do
  arch arm: "arm64", intel: "amd64"
  version "0.1.0"
  sha256 arm: "...", intel: "..."
  url "https://github.com/infomosquitos/ticketmosquito/releases/download/v#{version}/ticketmosquito-darwin-#{arch}"
  # binary + postflight xattr + zap + caveats…
end
```

See live reference: [Casks/spacemosquito.rb](https://github.com/infomosquitos/homebrew-tap/blob/main/Casks/spacemosquito.rb).
