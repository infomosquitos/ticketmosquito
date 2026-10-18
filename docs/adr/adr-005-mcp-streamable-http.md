# ADR-005: MCP transport — Streamable HTTP

- **Status:** Accepted
- **Date:** 2026-10-18
- **Related:** SpaceMosquito ADR-017; [Configure MCP](../guides/configure-mcp.md)

## Context

Agents need read access to the local Jira catalog. MCP remote transport is Streamable HTTP (`2025-03-26+`). TicketMosquito already runs a local HTTP server for the session API on port `11043`.

## Decision

Serve MCP **only** via Streamable HTTP on **`POST /mcp`** in the same process as `serve`:

| Topic | Choice |
|-------|--------|
| Endpoint | `http://127.0.0.1:11043/mcp` |
| Responses | `application/json` JSON-RPC |
| Sessions | Stateless (no `MCP-Session-Id`) |
| Security | Reject non-loopback `Origin` when present |
| Tools (MVP) | `jira_search`, `jira_get_issue`, `jira_list_projects` — local catalog only |
| Advertised protocol | `2025-03-26` |

## Alternatives considered

| Option | Why not |
|--------|---------|
| stdio MCP | Does not fit `serve` / remote-client model |
| HTTP+SSE dual-stack | Deprecated; extra surface |
| Live Jira proxy tools | Out of MVP scope; offline mirror is the product |

## Consequences

- Clients must speak Streamable HTTP, not SSE-only configs.
- Crawl remains CLI/extension/HTTP session — not started via MCP.
