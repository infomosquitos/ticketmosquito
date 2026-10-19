package session

import (
	"encoding/json"
	"fmt"
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

// AsHeaders returns Cookie + X-Atlassian-Token headers for REST calls.
func (s *Session) AsHeaders() map[string]string {
	headers := make(map[string]string)
	if len(s.Cookies) > 0 {
		parts := make([]string, 0, len(s.Cookies))
		for _, c := range s.Cookies {
			parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))
		}
		headers["Cookie"] = strings.Join(parts, "; ")
	}
	headers["X-Atlassian-Token"] = "no-check"
	headers["Accept"] = "application/json"
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
		for _, c := range s.Cookies {
			req.AddCookie(&http.Cookie{
				Name: c.Name, Value: c.Value, Domain: c.Domain, Path: c.Path,
				Expires: time.Unix(c.Expires, 0), Secure: c.Secure, HttpOnly: c.HTTPOnly,
			})
		}
		req.Header.Set("X-Atlassian-Token", "no-check")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
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
				"cookies", len(s.Cookies),
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
	if preferServerProbes(flavor, pageURL) {
		return []validationProbe{server, cloud}
	}
	return []validationProbe{cloud, server}
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
	kind   validationKind
	name   string
	status int
	reason string
}

func classifyValidationResponse(resp *http.Response) validationOutcome {
	status := resp.StatusCode
	if status == http.StatusMovedPermanently ||
		status == http.StatusFound ||
		status == http.StatusSeeOther ||
		status == http.StatusTemporaryRedirect ||
		status == http.StatusPermanentRedirect {
		return validationOutcome{kind: validationSSOIntercept, status: status, reason: "redirect"}
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return validationOutcome{kind: validationUnauthorized, status: status}
	}
	if status == http.StatusNotFound {
		return validationOutcome{kind: validationNotFound, status: status}
	}
	if status != http.StatusOK {
		return validationOutcome{kind: validationSSOIntercept, status: status, reason: "non-200"}
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(ct), "application/json") {
		return validationOutcome{kind: validationSSOIntercept, status: status, reason: "non-json content-type"}
	}

	var myself map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&myself); err != nil {
		return validationOutcome{kind: validationSSOIntercept, status: status, reason: "json decode failed"}
	}
	name, ok := identityFromMyself(myself)
	if !ok {
		return validationOutcome{kind: validationSSOIntercept, status: status, reason: "missing displayName/username"}
	}
	return validationOutcome{kind: validationOK, name: name, status: status}
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
	return "", false
}
