package session

import (
	"testing"
	"time"

	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

const testKey = "0123456789abcdef0123456789abcdef"

func TestHostnameKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"https://Jira.Example.com/browse/X-1", "jira.example.com"},
		{"https://acme.atlassian.net/browse/K-1", "acme.atlassian.net"},
		{"https://jira.example.com:8443/browse/K-1", "jira.example.com:8443"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := HostnameKey(tc.in); got != tc.want {
			t.Errorf("HostnameKey(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestStore_UpsertTwoHosts(t *testing.T) {
	tmpFile := t.TempDir() + "/session.enc"
	store := NewStore(tmpFile, logging.Sugar{})

	a := &Session{
		SiteURL:    "https://jira-a.example.com/browse/A-1",
		Cookies:    []Cookie{{Name: "JSESSIONID", Value: "aaa", Domain: "jira-a.example.com"}},
		CapturedAt: time.Now(),
	}
	b := &Session{
		SiteURL:    "https://jira-b.example.com/browse/B-1",
		Cookies:    []Cookie{{Name: "JSESSIONID", Value: "bbb", Domain: "jira-b.example.com"}},
		CapturedAt: time.Now(),
	}

	if err := store.Upsert(a, testKey); err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(b, testKey); err != nil {
		t.Fatal(err)
	}

	gotA, err := store.GetForURL(testKey, "https://jira-a.example.com/browse/A-2")
	if err != nil {
		t.Fatal(err)
	}
	if gotA.Cookies[0].Value != "aaa" {
		t.Fatalf("host A cookies overwritten: %+v", gotA.Cookies)
	}

	gotB, err := store.GetForHostname(testKey, "jira-b.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if gotB.Cookies[0].Value != "bbb" {
		t.Fatalf("host B wrong: %+v", gotB.Cookies)
	}

	if store.Count(testKey) != 2 {
		t.Fatalf("count = %d, want 2", store.Count(testKey))
	}
}

func TestSession_AsHeaders(t *testing.T) {
	s := &Session{
		SiteURL: "https://jira.example.com/browse/X-1",
		Cookies: []Cookie{
			{Name: "a", Value: "1", Domain: "jira.example.com"},
			{Name: "b", Value: "2", Domain: "jira.example.com"},
			{Name: "noise", Value: "x", Domain: "other.example.com"},
		},
	}
	h := s.AsHeaders()
	if h["Cookie"] != "a=1; b=2" {
		t.Fatalf("Cookie = %q", h["Cookie"])
	}
	if h["User-Agent"] == "" {
		t.Fatal("expected User-Agent")
	}
	if h["X-Atlassian-Token"] != "no-check" {
		t.Fatalf("missing X-Atlassian-Token")
	}
}
