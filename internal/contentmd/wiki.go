package contentmd

import (
	"regexp"
	"strings"
)

var wikiHeading = regexp.MustCompile(`(?m)^h([1-6])\.\s+(.*)$`)
var wikiBold = regexp.MustCompile(`\*([^*]+)\*`)
var wikiCode = regexp.MustCompile(`\{\{([^}]+)\}\}`)
var wikiLink = regexp.MustCompile(`\[([^]|]+)\|([^]]+)\]`)

// WikiToMarkdown applies a best-effort conversion of Jira wiki markup to Markdown.
// Incomplete by design — prefer rendered HTML when available.
func WikiToMarkdown(wiki string) string {
	s := strings.TrimSpace(wiki)
	if s == "" {
		return ""
	}
	s = wikiHeading.ReplaceAllStringFunc(s, func(src string) string {
		m := wikiHeading.FindStringSubmatch(src)
		if len(m) != 3 {
			return src
		}
		n := int(m[1][0] - '0')
		if n < 1 {
			n = 1
		}
		if n > 6 {
			n = 6
		}
		return strings.Repeat("#", n) + " " + m[2]
	})
	s = wikiLink.ReplaceAllString(s, "[$1]($2)")
	s = wikiCode.ReplaceAllString(s, "`$1`")
	s = wikiBold.ReplaceAllString(s, "**$1**")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return normalizeMarkdown(s)
}
