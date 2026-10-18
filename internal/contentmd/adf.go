package contentmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/rgonek/jira-adf-converter/converter"
)

var (
	adfConv     *converter.Converter
	adfConvOnce sync.Once
	adfConvErr  error
)

func adfConverter() (*converter.Converter, error) {
	adfConvOnce.Do(func() {
		adfConv, adfConvErr = converter.New(converter.Config{})
	})
	return adfConv, adfConvErr
}

// IsADF reports whether raw looks like an ADF document (type=doc).
func IsADF(raw json.RawMessage) bool {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return false
	}
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	return probe.Type == "doc"
}

// ADFToMarkdown converts Atlassian Document Format JSON to Markdown.
func ADFToMarkdown(raw json.RawMessage) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", nil
	}
	conv, err := adfConverter()
	if err != nil {
		return "", fmt.Errorf("adf converter: %w", err)
	}
	result, err := conv.Convert(raw)
	if err != nil {
		return "", fmt.Errorf("adf to markdown: %w", err)
	}
	return normalizeMarkdown(result.Markdown), nil
}

// ADFObjectToMarkdown marshals an ADF object (map/slice from JSON decode) to Markdown.
func ADFObjectToMarkdown(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	switch t := v.(type) {
	case string:
		// Sometimes APIs double-encode; try parse as JSON first.
		s := strings.TrimSpace(t)
		if s == "" {
			return "", nil
		}
		if strings.HasPrefix(s, "{") {
			return ADFToMarkdown(json.RawMessage(s))
		}
		return "", fmt.Errorf("description string is not ADF JSON")
	case json.RawMessage:
		return ADFToMarkdown(t)
	case map[string]any, []any:
		b, err := json.Marshal(t)
		if err != nil {
			return "", err
		}
		return ADFToMarkdown(b)
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return "", err
		}
		if !IsADF(b) {
			return "", fmt.Errorf("value is not ADF")
		}
		return ADFToMarkdown(b)
	}
}
