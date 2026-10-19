package session

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/infomosquitos/ticketmosquito/internal/jira"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// Cookie is a browser cookie captured for Jira auth.
type Cookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	Expires  int64  `json:"expires,omitempty"`
	Secure   bool   `json:"secure"`
	HTTPOnly bool   `json:"httpOnly"`
	SameSite string `json:"sameSite,omitempty"`
}

// SessionFlavor is Cloud vs Server/DC.
type SessionFlavor string

const (
	FlavorCloud  SessionFlavor = "cloud"
	FlavorServer SessionFlavor = "server"
)

// Session is a captured Jira browser session for one host.
type Session struct {
	SiteURL     string        `json:"jira_url"`
	Cookies     []Cookie      `json:"cookies"`
	CapturedAt  time.Time     `json:"captured_at"`
	ValidatedAt *time.Time    `json:"validated_at,omitempty"`
	Flavor      SessionFlavor `json:"flavor,omitempty"`
	log         logging.Sugar
}

// ValidationResult is the outcome of a live /myself probe.
type ValidationResult struct {
	Valid     bool          `json:"valid"`
	Message   string        `json:"message,omitempty"`
	ExpiresAt *int64        `json:"expires_at,omitempty"`
	Flavor    SessionFlavor `json:"flavor,omitempty"`
}

// IsExpired reports whether CapturedAt is older than maxAge.
func (s *Session) IsExpired(maxAge time.Duration) bool {
	return time.Since(s.CapturedAt) > maxAge
}

// SetLogger sets the session logger.
func (s *Session) SetLogger(l logging.Sugar) {
	s.log = l
}

// browserUserAgent mimics a normal desktop Chrome so corporate WAFs / Seraph
// do not treat Go's default User-Agent as a bot.
const browserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// AsHeaders returns Cookie + X-Atlassian-Token headers for REST calls.
func (s *Session) AsHeaders() map[string]string {
	headers := make(map[string]string)
	cookies := CookiesForURL(s.Cookies, s.SiteURL)
	if len(cookies) > 0 {
		parts := make([]string, 0, len(cookies))
		for _, c := range cookies {
			parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))
		}
		headers["Cookie"] = strings.Join(parts, "; ")
	}
	headers["X-Atlassian-Token"] = "no-check"
	headers["Accept"] = "application/json"
	headers["User-Agent"] = browserUserAgent
	if root := jira.BaseURL(s.SiteURL); root != "" {
		headers["Referer"] = root + "/"
		headers["Origin"] = root
	}
	return headers
}

// APIBase returns the REST API root for this session's flavor
// (e.g. https://host/rest/api/3 or …/rest/api/2).
func (s *Session) APIBase() string {
	root := jira.BaseURL(s.SiteURL)
	if root == "" {
		return ""
	}
	switch s.Flavor {
	case FlavorServer:
		return root + "/rest/api/2"
	default:
		return root + "/rest/api/3"
	}
}

// ValidateWithJira probes /myself endpoints to confirm cookies and detect flavor.
func (s *Session) ValidateWithJira(siteURL string, timeoutSeconds int) (*ValidationResult, error) {
	rootURL := jira.BaseURL(siteURL)
	if rootURL == "" {
		rootURL = jira.BaseURL(s.SiteURL)
	}
	if rootURL == "" {
		return &ValidationResult{Valid: false, Message: "no jira URL available"}, nil
	}
	if len(s.Cookies) == 0 {
		return &ValidationResult{Valid: false, Message: "no cookies in session"}, nil
	}

	pageURL := siteURL
	if pageURL == "" {
		pageURL = s.SiteURL
	}
	probes := validationProbes(s.Flavor, pageURL)
	applicable := CookiesForURL(s.Cookies, rootURL)

	if s.log.Enabled() {
		s.log.Infow("session validation starting",
			"root", rootURL,
			"page", pageURL,
			"flavor", string(s.Flavor),
			"cookies_stored", len(s.Cookies),
			"cookies_applicable", len(applicable),
			"cookie_meta", cookieDebugMeta(s.Cookies),
			"cookie_meta_applicable", cookieDebugMeta(applicable),
		)
	}

	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var lastErr error
	sawUnauthorized := false
	sawSSOIntercept := false

	for _, p := range probes {
		testURL := rootURL + p.path
		if s.log.Enabled() {
			s.log.Infow("probing session validation", "url", testURL, "flavor", p.flavor)
		}

		req, err := http.NewRequest(http.MethodGet, testURL, nil)
		if err != nil {
			continue
		}
		// Set Cookie as a raw header (same as AsHeaders). AddCookie sanitizes
		// values and can drop/mangle corporate SSO tokens.
		for k, v := range s.AsHeaders() {
			req.Header.Set(k, v)
		}
		if s.log.Enabled() {
			cookieHdr := req.Header.Get("Cookie")
			s.log.Infow("session validation request headers",
				"url", testURL,
				"cookie_header_len", len(cookieHdr),
				"cookie_names_in_header", cookieNamesFromHeader(cookieHdr),
				"accept", req.Header.Get("Accept"),
				"x_atlassian_token", req.Header.Get("X-Atlassian-Token"),
				"user_agent", req.Header.Get("User-Agent"),
				"referer", req.Header.Get("Referer"),
			)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if s.log.Enabled() {
				s.log.Infow("session validation probe transport error",
					"url", testURL, "err", err.Error())
			}
			continue
		}
		outcome := classifyValidationResponse(resp)
		resp.Body.Close()

		if s.log.Enabled() {
			s.log.Infow("session validation probe result",
				"url", testURL,
				"flavor", p.flavor,
				"kind", outcome.kind.String(),
				"status", outcome.status,
				"reason", outcome.reason,
				"cookies_applicable", len(applicable),
				"location", outcome.location,
				"www_authenticate", outcome.wwwAuth,
				"content_type", outcome.contentType,
				"set_cookie_names", setCookieNames(outcome.setCookie),
				"body_preview", outcome.bodyPreview,
			)
		}

		switch outcome.kind {
		case validationOK:
			now := time.Now()
			s.ValidatedAt = &now
			s.Flavor = p.flavor
			return &ValidationResult{
				Valid:   true,
				Message: fmt.Sprintf("authenticated as %s", outcome.name),
				Flavor:  p.flavor,
			}, nil
		case validationUnauthorized:
			sawUnauthorized = true
		case validationSSOIntercept:
			sawSSOIntercept = true
		case validationNotFound:
			// try next probe
		}
	}

	if sawUnauthorized {
		return &ValidationResult{Valid: false, Message: "authentication failed — session expired"}, nil
	}
	if sawSSOIntercept {
		return &ValidationResult{Valid: false, Message: "authentication failed — SSO or non-JSON response (session expired or incomplete)"}, nil
	}
	if lastErr != nil {
		return &ValidationResult{Valid: false, Message: fmt.Sprintf("request failed: %v", lastErr)}, nil
	}
	return &ValidationResult{Valid: false, Message: "jira API not found (404) at probed endpoints"}, nil
}

type validationProbe struct {
	path   string
	flavor SessionFlavor
}

func validationProbes(flavor SessionFlavor, pageURL string) []validationProbe {
	cloud := validationProbe{"/rest/api/3/myself", FlavorCloud}
	server := validationProbe{"/rest/api/2/myself", FlavorServer}
	// Cookie-session check used by Jira Server/DC (returns {"name":...}).
	authSession := validationProbe{"/rest/auth/1/session", FlavorServer}
	if preferServerProbes(flavor, pageURL) {
		return []validationProbe{server, authSession, cloud}
	}
	return []validationProbe{cloud, server, authSession}
}

func preferServerProbes(flavor SessionFlavor, pageURL string) bool {
	if flavor == FlavorServer {
		return true
	}
	if flavor == FlavorCloud {
		return false
	}
	u := strings.ToLower(pageURL)
	if strings.Contains(u, "atlassian.net") {
		return false
	}
	if parsed, err := url.Parse(pageURL); err == nil && parsed.Host != "" && !strings.Contains(parsed.Host, "atlassian.net") {
		return true
	}
	return false
}

type validationKind int

const (
	validationOK validationKind = iota
	validationUnauthorized
	validationSSOIntercept
	validationNotFound
)

func (k validationKind) String() string {
	switch k {
	case validationOK:
		return "ok"
	case validationUnauthorized:
		return "unauthorized"
	case validationSSOIntercept:
		return "sso_or_non_json"
	case validationNotFound:
		return "not_found"
	default:
		return fmt.Sprintf("kind_%d", int(k))
	}
}

type validationOutcome struct {
	kind        validationKind
	name        string
	status      int
	reason      string
	location    string
	wwwAuth     string
	contentType string
	setCookie   []string
	bodyPreview string
}

func classifyValidationResponse(resp *http.Response) validationOutcome {
	status := resp.StatusCode
	ct := resp.Header.Get("Content-Type")
	loc := resp.Header.Get("Location")
	www := resp.Header.Get("WWW-Authenticate")
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	base := validationOutcome{
		status:      status,
		location:    loc,
		wwwAuth:     www,
		contentType: ct,
		setCookie:   resp.Header.Values("Set-Cookie"),
		bodyPreview: sanitizePreview(body, 240),
	}

	if status == http.StatusMovedPermanently ||
		status == http.StatusFound ||
		status == http.StatusSeeOther ||
		status == http.StatusTemporaryRedirect ||
		status == http.StatusPermanentRedirect {
		base.kind = validationSSOIntercept
		base.reason = "redirect"
		return base
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		base.kind = validationUnauthorized
		return base
	}
	if status == http.StatusNotFound {
		base.kind = validationNotFound
		return base
	}
	if status != http.StatusOK {
		base.kind = validationSSOIntercept
		base.reason = "non-200"
		return base
	}

	if !strings.Contains(strings.ToLower(ct), "application/json") {
		base.kind = validationSSOIntercept
		base.reason = "non-json content-type"
		return base
	}

	var myself map[string]interface{}
	if err := json.Unmarshal(body, &myself); err != nil {
		base.kind = validationSSOIntercept
		base.reason = "json decode failed"
		return base
	}
	name, ok := identityFromMyself(myself)
	if !ok {
		base.kind = validationSSOIntercept
		base.reason = "missing displayName/username"
		return base
	}
	base.kind = validationOK
	base.name = name
	return base
}

func sanitizePreview(body []byte, n int) string {
	if len(body) == 0 {
		return ""
	}
	if len(body) > n {
		body = body[:n]
	}
	s := string(body)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.TrimSpace(s)
}

func cookieDebugMeta(cookies []Cookie) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(cookies))
	for _, c := range cookies {
		out = append(out, map[string]interface{}{
			"name":     c.Name,
			"domain":   c.Domain,
			"path":     c.Path,
			"secure":   c.Secure,
			"httpOnly": c.HTTPOnly,
			"sameSite": c.SameSite,
			"valueLen": len(c.Value),
			"expires":  c.Expires,
		})
	}
	return out
}

func cookieNamesFromHeader(header string) []string {
	if header == "" {
		return nil
	}
	parts := strings.Split(header, ";")
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		name, _, _ := strings.Cut(p, "=")
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func setCookieNames(values []string) []string {
	names := make([]string, 0, len(values))
	for _, v := range values {
		name, _, _ := strings.Cut(v, "=")
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func identityFromMyself(myself map[string]interface{}) (string, bool) {
	if name, ok := myself["displayName"].(string); ok {
		if name = strings.TrimSpace(name); name != "" {
			return name, true
		}
	}
	if name, ok := myself["username"].(string); ok {
		if name = strings.TrimSpace(name); name != "" {
			return name, true
		}
	}
	// /rest/auth/1/session and some Server payloads use "name".
	if name, ok := myself["name"].(string); ok {
		if name = strings.TrimSpace(name); name != "" {
			return name, true
		}
	}
	return "", false
}

// CookiesForURL returns cookies whose Domain would be sent to targetURL.
// Drops unrelated sibling-host cookies (e.g. portal.example.com NetScaler jars
// captured while walking example.com).
func CookiesForURL(cookies []Cookie, targetURL string) []Cookie {
	host := HostnameKey(targetURL)
	if host == "" {
		return append([]Cookie(nil), cookies...)
	}
	// HostnameKey may include :port — cookie domains usually do not.
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.ToLower(host)
	out := make([]Cookie, 0, len(cookies))
	for _, c := range cookies {
		if cookieDomainMatchesHost(c.Domain, host) {
			out = append(out, c)
		}
	}
	return out
}

func cookieDomainMatchesHost(cookieDomain, host string) bool {
	dom := strings.ToLower(strings.TrimSpace(cookieDomain))
	if dom == "" {
		return false
	}
	dom = strings.TrimPrefix(dom, ".")
	if dom == host {
		return true
	}
	return strings.HasSuffix(host, "."+dom)
}
