package contentmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// BodyInput is one description or comment body with optional sources.
// Conversion prefers ADF → wiki → rendered HTML (see ConvertBody).
type BodyInput struct {
	// ADF is Cloud description/comment body as raw ADF JSON (type=doc).
	ADF json.RawMessage
	// Wiki is Server/DC wiki markup string.
	Wiki string
	// HTML is rendered HTML from renderedFields / renderedBody (fallback).
	HTML string
}

// CommentInput is one issue comment for concatenation into searchable Markdown.
type CommentInput struct {
	Author  string
	Created time.Time
	Body    BodyInput
}

// IssueContentInput is description + ordered comments for one issue.
type IssueContentInput struct {
	Description BodyInput
	Comments    []CommentInput
}

// Result is the Markdown catalog body plus the primary body_format tag.
type Result struct {
	Markdown   string
	BodyFormat BodyFormat
}

// ConvertBody converts a single body using the dual path:
// structured (ADF, then wiki) primary; rendered HTML fallback when structured
// fails or is empty.
func ConvertBody(in BodyInput) (md string, format BodyFormat, err error) {
	if IsADF(in.ADF) {
		md, err = ADFToMarkdown(in.ADF)
		if err == nil && strings.TrimSpace(md) != "" {
			return md, FormatADF, nil
		}
		// structured failed/empty → try HTML below, then wiki
	}

	if strings.TrimSpace(in.Wiki) != "" {
		md = WikiToMarkdown(in.Wiki)
		if strings.TrimSpace(md) != "" {
			if strings.TrimSpace(in.HTML) != "" {
				htmlMD, htmlErr := HTMLToMarkdown(in.HTML)
				if htmlErr == nil && strings.TrimSpace(htmlMD) != "" && len(htmlMD) >= len(md) {
					return htmlMD, FormatRendered, nil
				}
			}
			return md, FormatWiki, nil
		}
	}

	if strings.TrimSpace(in.HTML) != "" {
		md, err = HTMLToMarkdown(in.HTML)
		if err != nil {
			return "", FormatEmpty, err
		}
		if strings.TrimSpace(md) != "" {
			return md, FormatRendered, nil
		}
	}

	return "", FormatEmpty, nil
}

// RenderIssue builds searchable Markdown for an issue (description + comments).
// BodyFormat reflects the description’s conversion path (comments may differ).
func RenderIssue(in IssueContentInput) (Result, error) {
	var b strings.Builder

	descMD, descFmt, err := ConvertBody(in.Description)
	if err != nil {
		return Result{}, fmt.Errorf("description: %w", err)
	}

	if strings.TrimSpace(descMD) != "" {
		b.WriteString(descMD)
		b.WriteString("\n")
	}

	if len(in.Comments) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("## Comments\n")
		for _, c := range in.Comments {
			cmd, _, cerr := ConvertBody(c.Body)
			if cerr != nil {
				return Result{}, fmt.Errorf("comment: %w", cerr)
			}
			b.WriteString("\n### ")
			b.WriteString(commentHeading(c))
			b.WriteString("\n\n")
			if strings.TrimSpace(cmd) != "" {
				b.WriteString(cmd)
				b.WriteString("\n")
			} else {
				b.WriteString("_(empty)_\n")
			}
		}
	}

	out := normalizeMarkdown(b.String())
	return Result{Markdown: out, BodyFormat: descFmt}, nil
}

func commentHeading(c CommentInput) string {
	author := strings.TrimSpace(c.Author)
	if author == "" {
		author = "Unknown"
	}
	if c.Created.IsZero() {
		return author
	}
	return fmt.Sprintf("%s — %s", author, c.Created.UTC().Format(time.RFC3339))
}
