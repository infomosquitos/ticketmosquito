package jira

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	issueKeyRe      = regexp.MustCompile(`(?i)^([A-Z][A-Z0-9]+)-(\d+)$`)
	browseIssueRe   = regexp.MustCompile(`(?i)/browse/([A-Z][A-Z0-9]+-\d+)(?:/|$)`)
	browseProjectRe = regexp.MustCompile(`(?i)/browse/([A-Z][A-Z0-9]+)(?:/|$)`)
	projectsRe      = regexp.MustCompile(`(?i)/projects?/([A-Z][A-Z0-9]+)(?:/|$)`)
	bareKeyRe       = regexp.MustCompile(`(?i)^[A-Z][A-Z0-9]+$`)
)

// ParseProjectRef extracts a project key from a URL or bare key.
// issueKey is set when the input is an issue URL or KEY-N.
// Project landing URLs like /browse/PROJ (no issue number) are accepted —
// that is the form BrowseURL writes into the projects table.
func ParseProjectRef(raw string) (projectKey, issueKey string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("empty project reference")
	}

	if m := issueKeyRe.FindStringSubmatch(raw); m != nil {
		return strings.ToUpper(m[1]), strings.ToUpper(m[0]), nil
	}
	if bareKeyRe.MatchString(raw) && !strings.ContainsAny(raw, "/.") {
		return strings.ToUpper(raw), "", nil
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", "", fmt.Errorf("invalid project url or key %q", raw)
	}
	path := u.Path
	if m := browseIssueRe.FindStringSubmatch(path); m != nil {
		full := strings.ToUpper(m[1])
		return strings.Split(full, "-")[0], full, nil
	}
	if m := browseProjectRe.FindStringSubmatch(path); m != nil {
		return strings.ToUpper(m[1]), "", nil
	}
	if m := projectsRe.FindStringSubmatch(path); m != nil {
		return strings.ToUpper(m[1]), "", nil
	}
	if q := u.Query().Get("projectKey"); q != "" {
		return strings.ToUpper(q), "", nil
	}
	return "", "", fmt.Errorf("could not extract project key from %q", raw)
}

// BrowseURL builds a project landing URL from site base and key.
func BrowseURL(siteBase, projectKey string) string {
	base := strings.TrimRight(BaseURL(siteBase), "/")
	if base == "" {
		base = strings.TrimRight(siteBase, "/")
	}
	return fmt.Sprintf("%s/browse/%s", base, strings.ToUpper(projectKey))
}
