# ADR-003: Hybrid browser-session authentication

- **Status:** Accepted
- **Date:** 2026-10-18
- **Related:** SpaceMosquito ADR-002 / ADR-005

## Context

Jira Cloud and Server/DC often require interactive SSO. API tokens are optional later; MVP needs the same cookie-capture path as SpaceMosquito.

## Decision

1. Browser extension captures Atlassian-relevant cookies from a Jira tab.
2. `POST /api/session` stores them in an AES-256-GCM encrypted multi-host `session.enc` (key in `config.yaml`).
3. Backend validates via `GET …/rest/api/3/myself` (Cloud) or `…/rest/api/2/myself` (Server/DC) and records flavor.
4. Crawl/API clients reuse cookies as `Cookie` + `X-Atlassian-Token: no-check`.

## Consequences

- Separate Ticket Mosquito extension (Chrome + Firefox), default backend `http://127.0.0.1:11043`.
- Session expiry requires re-capture; HTTP API remains local/unauthenticated for MVP.
