package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// ErrUnauthorized is returned when Jira rejects the request (HTTP 401/403).
var ErrUnauthorized = errors.New("jira unauthorized")

// IsUnauthorized reports whether err is (or wraps) ErrUnauthorized.
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// Client talks to Jira REST (Cloud v3 or Server/DC v2) using caller-supplied auth headers.
type Client struct {
	http    *http.Client
	apiBase string
	server  bool
	headers map[string]string
	log     logging.Sugar
}

// ClientConfig configures a REST client.
type ClientConfig struct {
	APIBase string
	Server  bool // true → /rest/api/2/search; false → /rest/api/3/search/jql
	Headers map[string]string
}

// NewClient builds a REST client.
func NewClient(cfg ClientConfig, log logging.Sugar) (*Client, error) {
	base := strings.TrimRight(cfg.APIBase, "/")
	if base == "" {
		return nil, fmt.Errorf("API base is required")
	}
	headers := cfg.Headers
	if headers == nil {
		headers = map[string]string{}
	}
	return &Client{
		http: &http.Client{
			Timeout: 60 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		apiBase: base,
		server:  cfg.Server,
		headers: headers,
		log:     log,
	}, nil
}

// IssueRef is a lightweight discovery hit.
type IssueRef struct {
	ID      string
	Key     string
	Updated time.Time
}

// IssueDetail is a full issue payload for crawl persistence.
type IssueDetail struct {
	ID              string
	Key             string
	Summary         string
	Updated         time.Time
	FieldsJSON      json.RawMessage
	DescriptionADF  json.RawMessage
	DescriptionWiki string
	DescriptionHTML string
	Attachments     []AttachmentMeta
	Raw             json.RawMessage
}

// AttachmentMeta is attachment metadata from fields.attachment[].
type AttachmentMeta struct {
	ID       string
	Filename string
	MIMEType string
	Content  string
	Size     int64
}

// UnmarshalJSON accepts id as string or number (Cloud often sends numeric ids).
func (a *AttachmentMeta) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID       json.RawMessage `json:"id"`
		Filename string          `json:"filename"`
		MIMEType string          `json:"mimeType"`
		Content  string          `json:"content"`
		Size     int64           `json:"size"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	a.Filename = raw.Filename
	a.MIMEType = raw.MIMEType
	a.Content = raw.Content
	a.Size = raw.Size
	if len(raw.ID) == 0 || string(raw.ID) == "null" {
		return nil
	}
	var asStr string
	if err := json.Unmarshal(raw.ID, &asStr); err == nil {
		a.ID = asStr
		return nil
	}
	var asNum json.Number
	if err := json.Unmarshal(raw.ID, &asNum); err == nil {
		a.ID = asNum.String()
		return nil
	}
	// Fallback: trim quotes/spaces from raw
	a.ID = strings.Trim(string(raw.ID), `"`)
	return nil
}

// APIBase returns the REST root (e.g. …/rest/api/3).
func (c *Client) APIBase() string { return c.apiBase }

// AuthHeaders returns a copy of the headers used for REST calls.
func (c *Client) AuthHeaders() map[string]string {
	out := make(map[string]string, len(c.headers))
	for k, v := range c.headers {
		out[k] = v
	}
	return out
}

// CommentDetail is one comment with body sources.
type CommentDetail struct {
	ID       string
	Author   string
	Created  time.Time
	BodyADF  json.RawMessage
	BodyWiki string
	BodyHTML string
}

// DiscoverProgress reports discovery pagination. totalHint is Jira's reported
// total when known (Server/DC); 0 means unknown (Cloud token pages).
type DiscoverProgress func(foundSoFar, totalHint int)

// SearchProjectIssues lists all issues in a project (id, key, updated).
// onProgress is optional and may be called after each search page.
func (c *Client) SearchProjectIssues(ctx context.Context, projectKey string, onProgress DiscoverProgress) ([]IssueRef, error) {
	jql := fmt.Sprintf(`project = "%s" ORDER BY key ASC`, strings.ToUpper(projectKey))
	if c.server {
		return c.searchServer(ctx, jql, onProgress)
	}
	return c.searchCloud(ctx, jql, onProgress)
}

func (c *Client) searchCloud(ctx context.Context, jql string, onProgress DiscoverProgress) ([]IssueRef, error) {
	var out []IssueRef
	token := ""
	for {
		body := map[string]any{
			"jql":        jql,
			"maxResults": 100,
			"fields":     []string{"updated"},
		}
		if token != "" {
			body["nextPageToken"] = token
		}
		var resp struct {
			Issues []struct {
				ID     string `json:"id"`
				Key    string `json:"key"`
				Fields struct {
					Updated string `json:"updated"`
				} `json:"fields"`
			} `json:"issues"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := c.postJSON(ctx, c.apiBase+"/search/jql", body, &resp); err != nil {
			return nil, err
		}
		for _, iss := range resp.Issues {
			updated, _ := ParseTime(iss.Fields.Updated)
			out = append(out, IssueRef{ID: iss.ID, Key: iss.Key, Updated: updated})
		}
		if onProgress != nil {
			onProgress(len(out), 0)
		}
		if resp.NextPageToken == "" {
			break
		}
		token = resp.NextPageToken
	}
	return out, nil
}

func (c *Client) searchServer(ctx context.Context, jql string, onProgress DiscoverProgress) ([]IssueRef, error) {
	var out []IssueRef
	start := 0
	const page = 50
	for {
		body := map[string]any{
			"jql":        jql,
			"startAt":    start,
			"maxResults": page,
			"fields":     []string{"updated"},
		}
		var resp struct {
			StartAt    int `json:"startAt"`
			MaxResults int `json:"maxResults"`
			Total      int `json:"total"`
			Issues     []struct {
				ID     string `json:"id"`
				Key    string `json:"key"`
				Fields struct {
					Updated string `json:"updated"`
				} `json:"fields"`
			} `json:"issues"`
		}
		if err := c.postJSON(ctx, c.apiBase+"/search", body, &resp); err != nil {
			return nil, err
		}
		for _, iss := range resp.Issues {
			updated, _ := ParseTime(iss.Fields.Updated)
			out = append(out, IssueRef{ID: iss.ID, Key: iss.Key, Updated: updated})
		}
		if onProgress != nil {
			onProgress(len(out), resp.Total)
		}
		start += len(resp.Issues)
		if start >= resp.Total || len(resp.Issues) == 0 {
			break
		}
	}
	return out, nil
}

// GetIssue fetches one issue with renderedFields expand.
func (c *Client) GetIssue(ctx context.Context, issueKeyOrID string) (*IssueDetail, error) {
	u := fmt.Sprintf("%s/issue/%s?expand=renderedFields", c.apiBase, url.PathEscape(issueKeyOrID))
	raw, err := c.getBytes(ctx, u)
	if err != nil {
		return nil, err
	}
	return parseIssueDetail(raw)
}

// ListComments paginates all comments for an issue.
func (c *Client) ListComments(ctx context.Context, issueKeyOrID string) ([]CommentDetail, error) {
	var out []CommentDetail
	start := 0
	const page = 100
	for {
		u := fmt.Sprintf("%s/issue/%s/comment?startAt=%d&maxResults=%d&expand=renderedBody&orderBy=created",
			c.apiBase, url.PathEscape(issueKeyOrID), start, page)
		var resp struct {
			StartAt    int               `json:"startAt"`
			MaxResults int               `json:"maxResults"`
			Total      int               `json:"total"`
			Comments   []json.RawMessage `json:"comments"`
		}
		if err := c.getJSON(ctx, u, &resp); err != nil {
			return nil, err
		}
		for _, raw := range resp.Comments {
			cd, err := parseComment(raw)
			if err != nil {
				return nil, err
			}
			out = append(out, cd)
		}
		start += len(resp.Comments)
		if start >= resp.Total || len(resp.Comments) == 0 {
			break
		}
	}
	return out, nil
}

func parseIssueDetail(raw []byte) (*IssueDetail, error) {
	var envelope struct {
		ID             string          `json:"id"`
		Key            string          `json:"key"`
		Fields         json.RawMessage `json:"fields"`
		RenderedFields struct {
			Description string `json:"description"`
		} `json:"renderedFields"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode issue: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(envelope.Fields, &fields); err != nil {
		return nil, fmt.Errorf("decode fields: %w", err)
	}

	d := &IssueDetail{
		ID:              envelope.ID,
		Key:             envelope.Key,
		FieldsJSON:      envelope.Fields,
		DescriptionHTML: envelope.RenderedFields.Description,
		Raw:             raw,
	}
	if s, ok := fields["summary"]; ok {
		_ = json.Unmarshal(s, &d.Summary)
	}
	if u, ok := fields["updated"]; ok {
		var us string
		_ = json.Unmarshal(u, &us)
		d.Updated, _ = ParseTime(us)
	}
	if desc, ok := fields["description"]; ok && len(desc) > 0 && string(desc) != "null" {
		trim := bytes.TrimSpace(desc)
		if len(trim) > 0 && trim[0] == '{' {
			d.DescriptionADF = append(json.RawMessage(nil), desc...)
		} else {
			var wiki string
			if err := json.Unmarshal(desc, &wiki); err == nil {
				d.DescriptionWiki = wiki
			}
		}
	}
	if att, ok := fields["attachment"]; ok && string(att) != "null" {
		_ = json.Unmarshal(att, &d.Attachments)
	}
	return d, nil
}

func parseComment(raw []byte) (CommentDetail, error) {
	var c struct {
		ID     string `json:"id"`
		Author struct {
			DisplayName string `json:"displayName"`
			Name        string `json:"name"`
		} `json:"author"`
		Created      string          `json:"created"`
		Body         json.RawMessage `json:"body"`
		RenderedBody string          `json:"renderedBody"`
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return CommentDetail{}, err
	}
	cd := CommentDetail{
		ID:       c.ID,
		Author:   c.Author.DisplayName,
		BodyHTML: c.RenderedBody,
	}
	if cd.Author == "" {
		cd.Author = c.Author.Name
	}
	cd.Created, _ = ParseTime(c.Created)
	if len(c.Body) > 0 && string(c.Body) != "null" {
		trim := bytes.TrimSpace(c.Body)
		if len(trim) > 0 && trim[0] == '{' {
			cd.BodyADF = append(json.RawMessage(nil), c.Body...)
		} else {
			var wiki string
			if err := json.Unmarshal(c.Body, &wiki); err == nil {
				cd.BodyWiki = wiki
			}
		}
	}
	return cd, nil
}

func (c *Client) postJSON(ctx context.Context, rawURL string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)
	return c.doJSON(req, out)
}

func (c *Client) getJSON(ctx context.Context, rawURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	c.applyAuth(req)
	return c.doJSON(req, out)
}

func (c *Client) getBytes(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	c.applyAuth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited (429)")
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w (%d)", ErrUnauthorized, resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: HTTP %d: %s", rawURL, resp.StatusCode, truncate(string(data), 200))
	}
	return data, nil
}

func (c *Client) doJSON(req *http.Request, out any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("rate limited (429)")
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("%w (%d)", ErrUnauthorized, resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: HTTP %d: %s", req.Method, req.URL.String(), resp.StatusCode, truncate(string(data), 200))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}

func (c *Client) applyAuth(req *http.Request) {
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
