# ADR-001: Go backend language

- **Status:** Accepted
- **Date:** 2026-10-18
- **Related:** SpaceMosquito ADR-011 (same choice)

## Context

TicketMosquito mirrors SpaceMosquito: single local binary for CLI, HTTP API, crawl, and MCP.

## Decision

Implement the backend in **Go**, distributed as a single binary.

## Consequences

- Shared patterns with SpaceMosquito (selective paste).
- CGO-free SQLite via `modernc.org/sqlite`.
