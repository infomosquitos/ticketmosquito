package jira

import (
	"strings"
	"time"
)

// ParseTime parses Jira REST timestamps (often +0000 without colon).
func ParseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000+0000",
		"2006-01-02T15:04:05-0700",
		"2006-01-02T15:04:05+0000",
	}
	var last error
	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t.UTC(), nil
		}
		last = err
	}
	// Normalize +0000 → +00:00
	if len(s) >= 5 {
		tail := s[len(s)-5:]
		if (tail[0] == '+' || tail[0] == '-') && tail[3] != ':' {
			s2 := s[:len(s)-5] + string(tail[0]) + tail[1:3] + ":" + tail[3:]
			if t, err := time.Parse(time.RFC3339Nano, s2); err == nil {
				return t.UTC(), nil
			}
			if t, err := time.Parse(time.RFC3339, s2); err == nil {
				return t.UTC(), nil
			}
		}
	}
	return time.Time{}, last
}
