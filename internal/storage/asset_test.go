package storage

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testDownloader(client *http.Client) *AssetDownloader {
	return &AssetDownloader{
		client:     client,
		maxRetries: 1,
		retryDelay: time.Millisecond,
	}
}

func TestStableAttachmentName(t *testing.T) {
	got := StableAttachmentName("10001", "report.pdf")
	if got != "10001_report.pdf" {
		t.Fatalf("got %q", got)
	}
	got = StableAttachmentName("9", "../../etc/passwd")
	if strings.Contains(got, "..") || strings.Contains(got, "/") {
		t.Fatalf("unsafe name: %q", got)
	}
	if !strings.HasPrefix(got, "9_") {
		t.Fatalf("got %q", got)
	}
}

func TestResolveAttachmentURL(t *testing.T) {
	abs := "https://acme.atlassian.net/rest/api/3/attachment/content/1"
	if got := ResolveAttachmentURL(abs, "https://acme.atlassian.net/rest/api/3", "1"); got != abs {
		t.Fatalf("got %q", got)
	}
	api := "https://acme.atlassian.net/rest/api/3"
	want := api + "/attachment/content/42"
	if got := ResolveAttachmentURL("", api, "42"); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	rel := "/secure/attachment/9/file.png"
	got := ResolveAttachmentURL(rel, "https://jira.example.com/jira/rest/api/2", "9")
	if got != "https://jira.example.com/jira/secure/attachment/9/file.png" {
		t.Fatalf("got %q", got)
	}
}

func TestAssetDownloader_DownloadAs_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "JSESSIONID=x" {
			t.Errorf("missing cookie, got %q", r.Header.Get("Cookie"))
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4"))
	}))
	defer srv.Close()

	d := testDownloader(srv.Client())
	d.SetAuthHeaders(map[string]string{"Cookie": "JSESSIONID=x"})
	dest := filepath.Join(t.TempDir(), "assets", "attachments", "1_doc.pdf")
	if err := d.DownloadAs(dest, srv.URL+"/content/1"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dest)
	if err != nil || string(b) != "%PDF-1.4" {
		t.Fatalf("file=%q err=%v", b, err)
	}
}

func TestAssetDownloader_DownloadAs_skipsExisting(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte("new"))
	}))
	defer srv.Close()

	d := testDownloader(srv.Client())
	dest := filepath.Join(t.TempDir(), "1_a.bin")
	if err := os.WriteFile(dest, []byte("cached"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := d.DownloadAs(dest, srv.URL+"/a"); err != nil {
		t.Fatal(err)
	}
	if hits != 0 {
		t.Fatalf("hits=%d want 0", hits)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "cached" {
		t.Fatalf("got %q", got)
	}
}

func TestAssetDownloader_DownloadAs_forceOverwrites(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte("forced"))
	}))
	defer srv.Close()

	d := testDownloader(srv.Client())
	d.SetForce(true)
	dest := filepath.Join(t.TempDir(), "1_a.bin")
	if err := os.WriteFile(dest, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := d.DownloadAs(dest, srv.URL+"/a"); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("hits=%d", hits)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "forced" {
		t.Fatalf("got %q", got)
	}
}

func TestAssetDownloader_DownloadAs_rejectsHTMLLoginPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body>login</body></html>`))
	}))
	defer srv.Close()

	d := testDownloader(srv.Client())
	dest := filepath.Join(t.TempDir(), "1_x.png")
	if err := d.DownloadAs(dest, srv.URL+"/x.png"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(dest); err == nil {
		t.Fatal("file should not exist")
	}
}

func TestRelativeToIssue(t *testing.T) {
	issue := filepath.Join(t.TempDir(), "DEMO", "DEMO-1")
	abs := filepath.Join(issue, "assets", "attachments", "1_a.pdf")
	rel := RelativeToIssue(issue, abs)
	if rel != "assets/attachments/1_a.pdf" {
		t.Fatalf("got %q", rel)
	}
}
