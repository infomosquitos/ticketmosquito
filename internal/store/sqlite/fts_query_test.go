package sqlite

import "testing"

func TestBuildFTSQuery_AND(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{"login bug", `"login" AND "bug"`},
		{"single", `"single"`},
		{`say "hi"`, `"say" AND """hi"""`},
		{"  spaced  words  ", `"spaced" AND "words"`},
	}
	for _, tc := range tests {
		if got := buildFTSQuery(tc.query); got != tc.want {
			t.Errorf("buildFTSQuery(%q) = %q, want %q", tc.query, got, tc.want)
		}
	}
}
