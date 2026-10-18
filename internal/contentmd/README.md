# contentmd

Converts Jira issue description + comments into searchable Markdown for the local catalog (`content.md` / DB `content`).

## body_format

Stored on the issue (DB column + `metadata.json`) so reindex can pick the right source later:

| Value | Meaning |
|-------|---------|
| `adf` | Primary conversion from Atlassian Document Format (Cloud) |
| `wiki` | Primary conversion from Server/DC wiki markup (best-effort) |
| `rendered` | Fallback from rendered HTML (`renderedFields` / `renderedBody`) |
| _(empty)_ | No body content |

## Conversion order (`ConvertBody`)

1. ADF JSON (`type: doc`) → Markdown via `github.com/rgonek/jira-adf-converter`
2. Else wiki markup → light Markdown heuristics (prefer HTML if richer)
3. Else rendered HTML → Markdown via `html-to-markdown`

## Issue Markdown layout

```markdown
<description markdown>

## Comments

### Alice — 2024-06-01T12:00:00Z

<comment markdown>
```

Use `RenderIssue` then `WriteFile(dir, result.Markdown)`.
