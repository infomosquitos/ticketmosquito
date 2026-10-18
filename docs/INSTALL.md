# Install

Install TicketMosquito from a GitHub Release when tags are published via CI.

Until the first release, use **[Install from source](INSTALL-FROM-SOURCE.md)**.

## Requirements

- A release asset for your OS/arch (`ticketmosquito-*`)
- Firefox or Chrome (for the Ticket Mosquito extension)

## Binary

1. Download the binary for your platform and optionally `SHA256SUMS` from the release.
2. Verify the checksum against `SHA256SUMS`.
3. Install onto your `PATH` (example for macOS/Linux):

```sh
chmod +x ticketmosquito-darwin-arm64
sudo mv ticketmosquito-darwin-arm64 /usr/local/bin/ticketmosquito
```

## Browser extension

Each release may include:

- `ticketmosquito-chrome-*.zip` — Chrome unpacked package
- `ticketmosquito-firefox-*.zip` — Firefox package (unsigned unless AMO signing is configured)

### Chrome

1. Unzip `ticketmosquito-chrome-*.zip` so `manifest.json` is at the top of the folder.
2. Open `chrome://extensions` → **Developer mode** → **Load unpacked** → select that folder.

### Firefox

For unsigned builds, use temporary load from a source build (`about:debugging`) —
see [Extension install](guides/extension-install.md). Signed `.xpi` installs (persistent)
require AMO credentials in the release pipeline.

## Next steps

```sh
ticketmosquito init
ticketmosquito serve
```

Then capture a Jira session and crawl — see the [README](../README.md).
