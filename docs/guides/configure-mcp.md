# Configure MCP (TicketMosquito)

TicketMosquito exposes **MCP Streamable HTTP** on the same process as `serve` (default `http://127.0.0.1:11043/mcp`). Tools read the **local** catalog only — they do not call live Jira.

## Start the server

```sh
ticketmosquito init    # once
ticketmosquito serve
```

Crawl at least one project first so search has data:

```sh
ticketmosquito crawl PROJ
```

## Tools

| Tool | Purpose |
|------|---------|
| `jira_search` | BM25/FTS search (`query`, optional `project`, `limit`) |
| `jira_get_issue` | Full Markdown + metadata by `issue_key` |
| `jira_list_projects` | List crawled projects |

## Client config sketch

Point your MCP client at Streamable HTTP:

- **URL:** `http://127.0.0.1:11043/mcp`
- **Transport:** Streamable HTTP (JSON-RPC `POST`, `Accept: application/json, text/event-stream`)

Example Cursor / Claude-style server entry (adjust to your client’s schema):

```json
{
  "mcpServers": {
    "ticketmosquito": {
      "url": "http://127.0.0.1:11043/mcp"
    }
  }
}
```

Some clients expect a command-based stdio server; TicketMosquito MVP is **HTTP-only** (same pattern as SpaceMosquito). Prefer a client that supports Streamable HTTP, or a small local proxy if needed.

## Also available over REST

- `GET /api/search?q=…&project=…`
- `GET /api/issues/{KEY}`
- `GET /api/projects`

CLI: `ticketmosquito search "…"`, `ticketmosquito get-issue KEY`.
