# Jira session capture and browser extension

- **Task ID:** `jira-session-and-extension`
- **Status:** done
- **Parent / blocked by:** parent: `ticketmosquito-mvp-epic`

## Problem

Crawls need authenticated access to Jira. SpaceMosquito’s hybrid cookie model works for Atlassian SSO; TicketMosquito needs the same for Jira hosts, plus a dedicated extension.

## Goal

Users can capture a Jira browser session into encrypted `session.enc` via a new Ticket Mosquito Firefox/Chrome extension talking to `http://127.0.0.1:11043`, with Cloud vs Server/DC flavor detection.

## Open questions

| # | Topic | Status | Decision |
|---|--------|--------|----------|
| 1 | Auth model | locked | Hybrid cookie capture → AES-GCM multi-host blob (SpaceMosquito ADR-002/005 pattern); rename site URL fields away from Confluence |
| 2 | Extension | locked | New separate extension (epic OQ8); selective paste of cookie filter + popup UX |
| 3 | Flavor probe | locked | Cloud: `GET /rest/api/3/myself`; DC: `GET /rest/api/2/myself` (+ context path detection) |

## Non-goals

- API token / basic auth as primary MVP path (optional later)
- Shared extension with SpaceMosquito
- Starting crawls from the extension (can be a thin follow-up; session POST is the MVP must)

## Implementation

Backend: paste-adapt `session/store.go`, validation probes for Jira, `POST /api/session` (+ status/clear). Extension: new `firefox-extension` / `chrome-extension` targeting Jira URL patterns, default API base `11043`.

## Done when

- [x] Extension sends cookies for a Jira host; server stores decryptable `session.enc`.
- [x] Flavor (Cloud vs DC) recorded and used for API base construction.
- [x] Session usable as HTTP Cookie + `X-Atlassian-Token: no-check` headers for a test probe (`/myself`).
- [x] Brief install notes for loading the extension unpacked / from source.

## Notes

See `spike-jira-api-content-model.md` (auth) and SpaceMosquito `internal/session`, `chrome-extension/`, `firefox-extension/`.

Shipped: `internal/session`, `internal/jira`, `internal/api`, ADR-003, `docs/guides/extension-install.md`, Chrome + Firefox extensions (session-only popup).
