package jira

import "testing"

func TestParseProjectRef(t *testing.T) {
	tests := []struct {
		in          string
		wantProject string
		wantIssue   string
	}{
		{"PROJ", "PROJ", ""},
		{"PROJ-12", "PROJ", "PROJ-12"},
		{"https://acme.atlassian.net/browse/ABC-9", "ABC", "ABC-9"},
		{"https://jira.example.com/browse/SK", "SK", ""},
		{"https://jira.example.com/jira/projects/ENG", "ENG", ""},
		{"https://jira.example.com/jira/browse/ENG-1", "ENG", "ENG-1"},
		{"https://jira.example.com/browse/ENG/", "ENG", ""},
	}
	for _, tc := range tests {
		p, i, err := ParseProjectRef(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if p != tc.wantProject || i != tc.wantIssue {
			t.Fatalf("%q → (%q,%q), want (%q,%q)", tc.in, p, i, tc.wantProject, tc.wantIssue)
		}
	}
}

func TestParseTime(t *testing.T) {
	tm, err := ParseTime("2024-06-01T12:00:00.000+0000")
	if err != nil {
		t.Fatal(err)
	}
	if tm.Year() != 2024 || tm.Month() != 6 || tm.Day() != 1 {
		t.Fatalf("got %v", tm)
	}
}
