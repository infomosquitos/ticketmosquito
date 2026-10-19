# Extension three-tab parity with SpaceMosquito

- **Task ID:** `extension-three-tab-parity`
- **Status:** done
- **Parent / blocked by:** — (post-MVP; builds on done `jira-session-and-extension`)

## Problem

Ticket Mosquito’s Chrome/Firefox extensions are **session-only** (flat capture / validate / clear + backend URL). SpaceMosquito’s extensions are the primary operator UI: three tabs (Page / Spaces / Settings), session chip, live page compare/refresh, space list with crawl progress, and per-space cron.

Users who know SpaceMosquito expect the same popup shape for Jira. Without it, crawl/refresh stays CLI-only and the extension cannot drive the product the epic diagram promised (`ext → api → scraper`).

## Goal

Ship Ticket Mosquito extensions (Chrome + Firefox) that **match SpaceMosquito’s popup design and operator functions**, remapped to Jira:

| SpaceMosquito | TicketMosquito |
|---------------|----------------|
| Tab **Page** | Tab **Issue** (OQ1) |
| Tab **Spaces** | Tab **Projects** |
| Tab **Settings** | Tab **Settings** (same) |
| Confluence space / page | Jira project / issue |
| Page version compare + refresh | Issue `updated` compare + single-issue refresh |
| Crawl space / crawl all / cancel | Crawl project / crawl all / cancel |
| Per-space cron panel | Per-project cron panel (in scope — OQ2/4) |
| Session chip → capture+validate | Same |
| Auto-renew session | Same |
| Backend unreachable banner + action gating | Same |

Visual: paste-adapt SpaceMosquito popup HTML/CSS structure (tabs, chip, rows, progress, cron panel, light/dark tokens) onto Ticket Mosquito branding (existing blue/teal mosquito assets — do **not** copy SM green brand).

## Open questions

All OQs locked — ready to implement (or split into backend-then-extension commits as convenient).

| # | Topic | Status | Decision |
|---|--------|--------|----------|
| 1 | Tab labels | locked | **B)** Issue / Projects / Settings |
| 2 | Backend API surface for this task | locked | **Full parity:** projects registry + async crawl jobs + issue compare/refresh + cron (SM-shaped `/api/projects|crawl|issues|cron`) |
| 3 | Projects registry | locked | Mirror SM spaces CRUD as `/api/projects` — add by URL/key, list with counts, delete (extends today’s list-from-DB) |
| 4 | Cron | locked | **Include** SM-parity cron in this task (no separate `jira-cron` deferral) |
| 5 | Issue tab compare metric | locked | Live vs stored **`updated`** timestamps (plus summary when available). Badge for `body_format` (`adf` / `wiki` / `rendered`), analogous to SM’s CSF/HTML engine chip |
| 6 | Content script / detector | locked | **A)** Content script + `issue-detector` (SM pattern); Chrome + Firefox. See Notes |
| 7 | Auto-renew | locked | **Port SM auto-renew as-is** for Jira hosts (Settings checkbox → background interval capture/validate) |
| 8 | Duplicate cookie / browser-probe cleanup | locked | **Do in this task** — dedupe captured cookies by name (keep best/last); remove or stop logging the always-401 extension `fetch` probe as a validation signal |
| 9 | Ordering vs TS7 / Homebrew | locked | **Do this task before `upgrade-extensions-typescript-7`.** Stay on TS ~5.9 so SpaceMosquito extension code can be paste-adapted without a TS7 toolchain fight (SM is also TS5-era). Homebrew tap spikes remain **independent** (orthogonal distribution work) |

## Non-goals

- Shared npm package or shared extension with SpaceMosquito
- Writing back to Jira (create/update issues)
- Chrome Web Store / AMO publish (install-from-`dist` remains)
- Changing MCP tool contracts
- Perfect pixel clone of SM brand colors

## Implementation

### Feature map (operator UX)

```mermaid
flowchart LR
  subgraph popup [Ticket Mosquito popup]
    issueTab[Issue tab]
    projectsTab[Projects tab]
    settingsTab[Settings tab]
    chip[Session chip]
  end

  subgraph api [ticketmosquito serve]
    session[/api/session*]
    projects[/api/projects*]
    crawl[/api/crawl*]
    issues[/api/issues* compare refresh]
    cron[/api/cron*]
  end

  chip --> session
  issueTab --> issues
  issueTab --> session
  projectsTab --> projects
  projectsTab --> crawl
  projectsTab --> cron
  settingsTab --> session
```

### Backend (required — OQ2–4 locked)

Today TM serve exposes session + catalog search/get/list-projects only. Crawl is **CLI** (`ticketmosquito crawl` / `crawl-issue`). This task adds HTTP parity with SpaceMosquito:

| SM endpoint | TM equivalent |
|-------------|----------------|
| `POST/GET/DELETE /api/spaces` | `/api/projects` CRUD (URL or key → project key; list with counts) |
| `POST /api/crawl`, status, cancel, list, cleanup | Same job model; `project_url` / `project_key` instead of `space_url` |
| `GET /api/pages/{id}`, `…/compare`, `POST …/refresh` | `GET /api/issues/{key}`, compare live vs stored `updated`, `POST …/refresh` → single-issue crawl |
| `/api/cron/*` | Per-project cron (full + incremental), config reload — paste-adapt SM cron for projects |

Reuse scraper package from CLI; do not reimplement crawl logic in the extension.

### Extension

1. Paste-adapt SM `popup.html` / `popup.css` / `popup.ts` structure → Jira naming and port `11043`.
2. Content script + detector for issue/project (Chrome + Firefox).
3. Background: capture-and-validate (chip), start/cancel crawl messages, **auto-renew** (OQ7 — port SM).
4. Session chip replaces separate Capture/Validate buttons; match SM clear/invalid flow.
5. Gate actions on backend up + session valid (SM `applyGating`).
6. Persist last tab in `chrome.storage.local` like SM.
7. Session polish (OQ8): dedupe cookies on capture; drop misleading browser-jar `fetch` probe (or demote to debug-only).

### Docs

Update `docs/guides/extension-install.md` for the three-tab UI; brief README note that crawl can be started from the extension once APIs land.

## Done when

- [x] Chrome + Firefox popups: tabs **Issue / Projects / Settings**; layout/CSS parity with SM structure under TM branding.
- [x] Session chip captures + validates; backend banner + gating behave like SM.
- [x] Issue tab: detect current Jira issue/project; show host, summary, live vs stored freshness; refresh one issue.
- [x] Projects tab: list/add/delete projects, start/stop crawl with progress, crawl-all, per-project cron panel.
- [x] Settings: backend URL + auto-renew (SM behavior).
- [x] Backend: projects CRUD, crawl jobs, issue compare/refresh, cron — with tests (httptest or integration).
- [x] Cookie dedupe + browser-probe cleanup (OQ8).
- [x] `make build-extensions` green on TS ~5.9; install guide updated.
- [x] All OQs locked (done).

## Notes

### Gap vs current Ticket Mosquito

| Area | SpaceMosquito | TicketMosquito today |
|------|---------------|----------------------|
| Popup | 3 tabs, ~900 LOC popup.ts | Flat session UI |
| Content script | Yes | No |
| Crawl HTTP API | Yes | No (CLI only) |
| Spaces/projects registry | Yes | List crawled projects only |
| Page/issue compare+refresh | Yes | Get issue Markdown only |
| Cron | Yes | No |
| Session API | Yes | Yes (working on hosted Jira) |

### Selective paste sources

- `SpaceMosquito/chrome-extension/popup/{popup.html,popup.css,popup.ts}`
- `SpaceMosquito/chrome-extension/{background.ts,content/,lib/api.ts}`
- Firefox twins under `firefox-extension/`
- Backend patterns: `spacemosquito/internal/api` crawl/spaces/cron/page_ops — rewrite for projects/issues

### Suggested phasing (OQ2 full parity)

1. Backend: project registry + crawl jobs + issue compare/refresh + cron.
2. Extension three-tab UI (Issue / Projects / Settings) wired to those APIs.
3. Extension polish: auto-renew, cookie dedupe, probe cleanup.

### Ordering note (OQ9)

Prefer **parity → then TS7**. SpaceMosquito extensions use TypeScript 5.x + webpack/ts-loader; TicketMosquito is pinned `~5.9.3` for the same reason. Pasting SM popup/background/content code first avoids adapting that paste to a new emit pipeline. Run `upgrade-extensions-typescript-7` afterward on the finished three-tab tree. Homebrew stays unrelated.

### OQ6 explained — content script / detector

SpaceMosquito’s **Issue/Page tab** does not only read `tab.url`. A **content script** runs in the Confluence tab and a **detector** parses the DOM/URL into structured fields (`spaceKey`, `pageId`, `pageTitle`, …). The popup asks the background, which asks the content script (`get-space-info`).

TicketMosquito needs the same pipeline for Jira so the Issue tab can show “you are on PROJ-123” and call compare/refresh with the right key — including cases where the URL alone is ambiguous.

**What the detector must produce (proposed):**

| Field | Source examples |
|-------|-----------------|
| `host` | `jira.example.com` / `*.atlassian.net` |
| `projectKey` | `/browse/PROJ-1` → `PROJ`; `/projects/PROJ/...`; board URLs |
| `issueKey` | `/browse/PROJ-1`; `selectedIssue=PROJ-1` on boards; issue view deep links |
| `summary` / title | Optional: tab title or DOM; live summary comes from API compare when session valid |

**URL shapes to cover (Cloud + Server/DC):**

- `/browse/ISSUE-KEY`
- `/projects/KEY/...` (project home without a specific issue)
- `/jira/software/c/projects/KEY/...` and `selectedIssue=`
- Optional context path: `/jira/browse/...`
- Not-on-Jira → empty info (Issue tab placeholder like SM’s “Not on a Confluence page”)

**OQ6 is choosing *how* we get that info:**

| Option | Meaning |
|--------|---------|
| **A (recommended)** | Content script + `issue-detector.ts` (SM pattern): works when URL is enough; can later read DOM for board `selectedIssue` if needed. Chrome + Firefox. |
| **B** | Background-only: parse `tab.url` in the service worker / popup — simpler, but weaker on boards where the issue is only in query/DOM. |
| **C** | Hybrid: URL parse first; content script only when `selectedIssue` / board paths need it. |

**Locked: A.**

### Related tasks

- Done: `jira-session-and-extension` (session MVP)
- Backlog: `upgrade-extensions-typescript-7` — **after** this task (OQ9)
- Backlog: `spike-extension-coinstall` / `homebrew-tap-distribution` (independent)

