package jira

import "testing"

func TestBaseURL(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"https://x.atlassian.net/browse/KEY-1", "https://x.atlassian.net"},
		{"https://jira.example.com/browse/KEY-1", "https://jira.example.com"},
		{"https://jira.example.com/jira/browse/KEY-1", "https://jira.example.com/jira"},
		{"https://jira.example.com/jira/projects/KEY", "https://jira.example.com/jira"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := BaseURL(tc.in); got != tc.want {
			t.Errorf("BaseURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
