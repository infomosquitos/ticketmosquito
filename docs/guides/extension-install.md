# Install Ticket Mosquito browser extension

Prerequisites: run `ticketmosquito serve` (default `http://127.0.0.1:11043`).

## Build from source

```sh
cd chrome-extension   # or firefox-extension
npm install
npm run build
```

Output is under `dist/`. Or from the repo root: `make build-extensions`.

## Chrome / Chromium

1. Open `chrome://extensions`
2. Enable **Developer mode**
3. **Load unpacked** → select `chrome-extension/dist`
4. Open a Jira tab and click the extension icon

## Firefox

1. Open `about:debugging#/runtime/this-firefox`
2. **Load Temporary Add-on…** → select `firefox-extension/dist/manifest.json`
3. Open a Jira tab and click the extension icon

## Using the popup

Three tabs (same shape as SpaceMosquito):

| Tab | Purpose |
|-----|---------|
| **Issue** | Current tab host / issue summary; live vs stored `updated`; refresh one issue |
| **Projects** | Add project URL/key, crawl / stop / crawl-all, per-project cron |
| **Settings** | Backend URL + auto-renew session |

**Session chip** (header): click to capture cookies from the active Jira tab and validate against `/myself`. Backend must be reachable (banner appears if not).

## Settings

Backend URL defaults to `http://127.0.0.1:11043`. Change it under **Settings** if you use another host/port.
