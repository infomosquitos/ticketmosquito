package contentmd

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
)

var multiNewline = regexp.MustCompile(`\n{3,}`)

var (
	mdConverter     *converter.Converter
	mdConverterOnce sync.Once
)

func markdownConverter() *converter.Converter {
	mdConverterOnce.Do(func() {
		mdConverter = converter.NewConverter(
			converter.WithPlugins(
				base.NewBasePlugin(),
				table.NewTablePlugin(),
			),
		)
	})
	return mdConverter
}

// HTMLToMarkdown converts rendered HTML into normalized Markdown.
func HTMLToMarkdown(html string) (string, error) {
	html = strings.TrimSpace(html)
	if html == "" {
		return "", nil
	}
	md, err := markdownConverter().ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("convert html to markdown: %w", err)
	}
	return normalizeMarkdown(md), nil
}

func normalizeMarkdown(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	s = strings.TrimSpace(strings.Join(lines, "\n"))
	s = multiNewline.ReplaceAllString(s, "\n\n")
	return s
}
