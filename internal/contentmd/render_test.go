package contentmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const sampleADF = `{
  "version": 1,
  "type": "doc",
  "content": [
    {
      "type": "heading",
      "attrs": {"level": 2},
      "content": [{"type": "text", "text": "Overview"}]
    },
    {
      "type": "paragraph",
      "content": [
        {"type": "text", "text": "Hello from "},
        {"type": "text", "text": "ADF", "marks": [{"type": "strong"}]}
      ]
    },
    {
      "type": "bulletList",
      "content": [
        {
          "type": "listItem",
          "content": [
            {"type": "paragraph", "content": [{"type": "text", "text": "one"}]}
          ]
        },
        {
          "type": "listItem",
          "content": [
            {"type": "paragraph", "content": [{"type": "text", "text": "two"}]}
          ]
        }
      ]
    }
  ]
}`

func TestADFToMarkdown(t *testing.T) {
	md, err := ADFToMarkdown(json.RawMessage(sampleADF))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "Overview") {
		t.Fatalf("missing heading: %q", md)
	}
	if !strings.Contains(md, "Hello from") {
		t.Fatalf("missing paragraph: %q", md)
	}
	if !strings.Contains(strings.ToLower(md), "adf") {
		t.Fatalf("missing bold text: %q", md)
	}
}

func TestRenderIssue_ADFAndComments(t *testing.T) {
	commentADF := `{
	  "version": 1,
	  "type": "doc",
	  "content": [
	    {"type": "paragraph", "content": [{"type": "text", "text": "Looks good"}]}
	  ]
	}`
	created := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	res, err := RenderIssue(IssueContentInput{
		Description: BodyInput{ADF: json.RawMessage(sampleADF)},
		Comments: []CommentInput{
			{
				Author:  "Alice",
				Created: created,
				Body:    BodyInput{ADF: json.RawMessage(commentADF)},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.BodyFormat != FormatADF {
		t.Fatalf("BodyFormat = %q, want adf", res.BodyFormat)
	}
	if !strings.Contains(res.Markdown, "## Comments") {
		t.Fatalf("missing comments section: %q", res.Markdown)
	}
	if !strings.Contains(res.Markdown, "### Alice — 2024-06-01T12:00:00Z") {
		t.Fatalf("missing comment heading: %q", res.Markdown)
	}
	if !strings.Contains(res.Markdown, "Looks good") {
		t.Fatalf("missing comment body: %q", res.Markdown)
	}
}

func TestConvertBody_HTMLFallback(t *testing.T) {
	md, format, err := ConvertBody(BodyInput{
		ADF:  json.RawMessage(`{"type":"doc","version":1,"content":[]}`),
		HTML: `<p>Rendered <strong>fallback</strong></p>`,
	})
	if err != nil {
		t.Fatal(err)
	}
	// empty ADF content → HTML fallback
	if format != FormatRendered {
		t.Fatalf("format = %q, want rendered", format)
	}
	if !strings.Contains(md, "fallback") {
		t.Fatalf("md = %q", md)
	}
}

func TestConvertBody_Wiki(t *testing.T) {
	md, format, err := ConvertBody(BodyInput{
		Wiki: "h2. Title\n\n*bold* and {{code}}",
	})
	if err != nil {
		t.Fatal(err)
	}
	if format != FormatWiki {
		t.Fatalf("format = %q", format)
	}
	if !strings.Contains(md, "## Title") {
		t.Fatalf("heading: %q", md)
	}
	if !strings.Contains(md, "**bold**") {
		t.Fatalf("bold: %q", md)
	}
	if !strings.Contains(md, "`code`") {
		t.Fatalf("code: %q", md)
	}
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	if err := WriteFile(dir, "# hi\n"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, ContentFileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "# hi\n" {
		t.Fatalf("got %q", b)
	}
}

func TestIsADF(t *testing.T) {
	if !IsADF(json.RawMessage(sampleADF)) {
		t.Fatal("expected true")
	}
	if IsADF(json.RawMessage(`{"type":"paragraph"}`)) {
		t.Fatal("expected false")
	}
}
