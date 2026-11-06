# Install

## macOS (Homebrew)

```sh
brew install --cask infomosquitos/tap/ticketmosquito
```

That installs the `ticketmosquito` binary on your `PATH`. Homebrew does **not** install the browser extension — see caveats after install, or [Extension install](guides/extension-install.md).

Tap: [`infomosquitos/homebrew-tap`](https://github.com/infomosquitos/homebrew-tap). The cask is updated automatically on each stable `v*` release.

## GitHub Release binary

1. Download the binary for your platform and optionally `SHA256SUMS` from the [Releases](https://github.com/infomosquitos/ticketmosquito/releases) page.
2. Verify the checksum against `SHA256SUMS`.
3. Install onto your `PATH` (example for macOS/Linux):

```sh
chmod +x ticketmosquito-darwin-arm64
sudo mv ticketmosquito-darwin-arm64 /usr/local/bin/ticketmosquito
```

From source: **[Install from source](INSTALL-FROM-SOURCE.md)**.

## Requirements

- A release asset for your OS/arch (`ticketmosquito-*`), or Homebrew on macOS
- Firefox or Chrome (for the Ticket Mosquito extension)

## Browser extension

Each release includes:

- `ticketmosquito-chrome-*.zip` — Chrome unpacked package
- `ticketmosquito-firefox-*.xpi` — Firefox add-on signed via AMO (unlisted)

### Chrome

1. Unzip `ticketmosquito-chrome-*.zip` so `manifest.json` is at the top of the folder.
2. Open `chrome://extensions` → **Developer mode** → **Load unpacked** → select that folder.

### Firefox

1. Download `ticketmosquito-firefox-*.xpi` from the release.
2. Open `about:addons` → gear icon → **Install Add-on From File…** → select the `.xpi`.

For local development builds, use temporary load instead —
see [Extension install](guides/extension-install.md).

## Next steps

```sh
ticketmosquito init
ticketmosquito serve
```

Then capture a Jira session and crawl — see the [README](../README.md).
