# Install Ticket Mosquito browser extension

Prerequisites: run `ticketmosquito serve` (default `http://127.0.0.1:11043`).

## Build from source

```sh
cd chrome-extension   # or firefox-extension
npm install
npm run build
```

Output is under `dist/`.

## Chrome / Chromium

1. Open `chrome://extensions`
2. Enable **Developer mode**
3. **Load unpacked** → select `chrome-extension/dist`
4. Open a Jira tab, click the extension, **Capture session**, then **Validate**

## Firefox

1. Open `about:debugging#/runtime/this-firefox`
2. **Load Temporary Add-on…** → select `firefox-extension/dist/manifest.json`
3. Open a Jira tab, capture and validate as above

## Settings

Backend URL defaults to `http://127.0.0.1:11043`. Change it in the popup if you use another host/port.
