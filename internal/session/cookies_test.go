package session

import "testing"

func TestCookiesForURL(t *testing.T) {
	cookies := []Cookie{
		{Name: "JSESSIONID", Domain: "jira.example.com"},
		{Name: "NSC_TMAS", Domain: "portal.example.com"},
		{Name: "parent", Domain: ".example.com"},
	}
	got := CookiesForURL(cookies, "https://jira.example.com/browse/X-1")
	if len(got) != 2 {
		t.Fatalf("got %d cookies: %+v", len(got), got)
	}
	names := map[string]bool{}
	for _, c := range got {
		names[c.Name] = true
	}
	if !names["JSESSIONID"] || !names["parent"] || names["NSC_TMAS"] {
		t.Fatalf("unexpected set: %v", names)
	}
}

func TestIdentityFromMyself_name(t *testing.T) {
	name, ok := identityFromMyself(map[string]interface{}{"name": "jdoe"})
	if !ok || name != "jdoe" {
		t.Fatalf("got %q ok=%v", name, ok)
	}
}
