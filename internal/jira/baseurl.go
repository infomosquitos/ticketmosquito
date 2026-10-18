// Package jira holds small shared helpers for Jira URLs.
package jira

import (
	"fmt"
	"net/url"
	"strings"
)

// browseMarkers are path prefixes that begin Jira browse/UI routes.
// Anything before the earliest marker is treated as the install context path
// (e.g. "/jira").
var browseMarkers = []string{
	"/browse/",
	"/projects/",
	"/secure/",
	"/jira/software/",
	"/jira/core/",
	"/rest/",
	"/servicedesk/",
}

// BaseURL returns the Jira site base for a browse or issue URL:
// scheme://host plus any context path before browse markers.
//
// Examples:
//   - https://x.atlassian.net/browse/KEY-1 → https://x.atlassian.net
//   - https://jira.example.com/browse/KEY-1 → https://jira.example.com
//   - https://jira.example.com/jira/browse/KEY-1 → https://jira.example.com/jira
func BaseURL(urlStr string) string {
	if urlStr == "" {
		return ""
	}
	u, err := url.Parse(urlStr)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	origin := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	ctx := contextPath(u.Path)
	if ctx == "" {
		return origin
	}
	return origin + ctx
}

func contextPath(path string) string {
	if path == "" || path == "/" {
		return ""
	}
	earliest := -1
	for _, marker := range browseMarkers {
		if i := strings.Index(path, marker); i >= 0 {
			if earliest < 0 || i < earliest {
				earliest = i
			}
		}
	}
	if earliest <= 0 {
		// No marker: if path looks like a single context segment, keep it.
		trimmed := strings.Trim(path, "/")
		if trimmed != "" && !strings.Contains(trimmed, "/") {
			return "/" + trimmed
		}
		return ""
	}
	return strings.TrimRight(path[:earliest], "/")
}
