package storage

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

const (
	defaultMaxRetries = 2
	defaultRetryDelay = 200 * time.Millisecond
)

// AssetDownloader fetches binary attachments with session auth.
type AssetDownloader struct {
	client      *http.Client
	log         logging.Sugar
	maxRetries  int
	retryDelay  time.Duration
	mu          sync.Mutex
	authHeaders map[string]string
	force       bool
}

// NewAssetDownloader creates a downloader with default retry policy.
func NewAssetDownloader(log logging.Sugar) *AssetDownloader {
	return &AssetDownloader{
		client: &http.Client{
			Timeout: 60 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		log:        log,
		maxRetries: defaultMaxRetries,
		retryDelay: defaultRetryDelay,
	}
}

// SetAuthHeaders sets Cookie / X-Atlassian-Token (and related) headers for every GET.
func (d *AssetDownloader) SetAuthHeaders(h map[string]string) {
	m := make(map[string]string, len(h))
	for k, v := range h {
		m[k] = v
	}
	d.mu.Lock()
	d.authHeaders = m
	d.mu.Unlock()
}

// SetForce controls whether existing on-disk assets are re-downloaded.
func (d *AssetDownloader) SetForce(force bool) {
	d.mu.Lock()
	d.force = force
	d.mu.Unlock()
}

// SetClientForTest swaps the HTTP client and disables retries for httptest tests.
func (d *AssetDownloader) SetClientForTest(c *http.Client) {
	d.client = c
	d.retryDelay = time.Millisecond
	d.maxRetries = 1
}

func (d *AssetDownloader) shouldSkip(path string) bool {
	d.mu.Lock()
	force := d.force
	d.mu.Unlock()
	if force {
		return false
	}
	fi, err := os.Stat(path)
	return err == nil && fi.Size() > 0
}

func looksLikeLoginPage(resp *http.Response) bool {
	return strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/html")
}

// DownloadAs fetches rawURL into destPath (creating parent dirs).
// Skips when the file already exists and is non-empty, unless force is set.
func (d *AssetDownloader) DownloadAs(destPath, rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("empty download URL")
	}
	if d.shouldSkip(destPath) {
		if d.log.Enabled() {
			d.log.Debugw("attachment already exists, skipping", "url", rawURL, "path", destPath)
		}
		return nil
	}

	var lastErr error
	for attempt := 0; attempt <= d.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * d.retryDelay)
		}

		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return err
		}
		d.mu.Lock()
		for k, v := range d.authHeaders {
			req.Header.Set(k, v)
		}
		d.mu.Unlock()

		resp, err := d.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request error: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			resp.Body.Close()
			return fmt.Errorf("download %s: HTTP %d (session unauthorized)", rawURL, resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}
		if looksLikeLoginPage(resp) {
			resp.Body.Close()
			return fmt.Errorf("download %s: got an HTML page (not the asset) — session likely missing or expired", rawURL)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("create asset dir: %w", err)
			continue
		}
		f, err := os.Create(destPath)
		if err != nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("create asset file: %w", err)
			continue
		}
		_, err = io.Copy(f, resp.Body)
		resp.Body.Close()
		f.Close()
		if err != nil {
			_ = os.Remove(destPath)
			lastErr = fmt.Errorf("write asset: %w", err)
			continue
		}
		if d.log.Enabled() {
			d.log.Infow("attachment downloaded", "url", rawURL, "path", destPath)
		}
		return nil
	}
	return fmt.Errorf("download %s: failed after %d attempts: %w", rawURL, d.maxRetries+1, lastErr)
}

var unsafeNameChars = regexp.MustCompile(`[^a-zA-Z0-9._\-+=@() ]+`)

// StableAttachmentName returns a deterministic filename: {id}_{sanitizedOriginal}.
func StableAttachmentName(id, filename string) string {
	base := filepath.Base(strings.TrimSpace(filename))
	if base == "" || base == "." || base == ".." {
		base = "attachment"
	}
	base = unsafeNameChars.ReplaceAllString(base, "_")
	base = strings.TrimSpace(base)
	if base == "" {
		base = "attachment"
	}
	// Collapse runs of underscores from sanitization.
	for strings.Contains(base, "__") {
		base = strings.ReplaceAll(base, "__", "_")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return base
	}
	// Keep id numeric/alphanumeric only for the prefix.
	cleanID := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			return r
		}
		return -1
	}, id)
	if cleanID == "" {
		return base
	}
	return cleanID + "_" + base
}

// AttachmentsDir returns issueDir/assets/attachments.
func AttachmentsDir(issueDir string) string {
	return filepath.Join(issueDir, "assets", "attachments")
}

// ImagesDir returns issueDir/assets/images.
func ImagesDir(issueDir string) string {
	return filepath.Join(issueDir, "assets", "images")
}

// RelativeToIssue returns path relative to issueDir when possible.
func RelativeToIssue(issueDir, absPath string) string {
	rel, err := filepath.Rel(issueDir, absPath)
	if err != nil {
		return absPath
	}
	return filepath.ToSlash(rel)
}

// ResolveAttachmentURL picks the download URL: metadata content URL, else Cloud REST path.
func ResolveAttachmentURL(contentURL, apiBase, attachmentID string) string {
	if contentURL != "" {
		if u, err := url.Parse(contentURL); err == nil && u.Scheme != "" && u.Host != "" {
			return contentURL
		}
		// Relative content path — join with site origin derived from apiBase.
		base := strings.TrimRight(apiBase, "/")
		// apiBase is …/rest/api/3 — strip to site root for relative /secure/… paths.
		if idx := strings.Index(base, "/rest/"); idx > 0 {
			origin := base[:idx]
			if strings.HasPrefix(contentURL, "/") {
				return origin + contentURL
			}
			return origin + "/" + contentURL
		}
		if strings.HasPrefix(contentURL, "/") {
			return base + contentURL
		}
		return contentURL
	}
	if attachmentID == "" || apiBase == "" {
		return ""
	}
	return strings.TrimRight(apiBase, "/") + "/attachment/content/" + url.PathEscape(attachmentID)
}
