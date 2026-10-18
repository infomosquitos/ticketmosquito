package contentmd

// BodyFormat records how issue Markdown was primarily produced.
// Stored in metadata / DB for later reindex routing.
type BodyFormat string

const (
	// FormatADF — Cloud description/comments converted from Atlassian Document Format.
	FormatADF BodyFormat = "adf"
	// FormatWiki — Server/DC wiki markup used as the structured source (often lightly normalized).
	FormatWiki BodyFormat = "wiki"
	// FormatRendered — Markdown from rendered HTML (API renderedFields / renderedBody or browser).
	FormatRendered BodyFormat = "rendered"
	// FormatEmpty — no description/comments content.
	FormatEmpty BodyFormat = ""
)

// ContentFileName is the on-disk markdown artifact under a saved issue directory.
const ContentFileName = "content.md"
