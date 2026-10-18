package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// Metadata is persisted as metadata.json beside content.md.
type Metadata struct {
	Key         string         `json:"key"`
	JiraID      string         `json:"jira_id"`
	Summary     string         `json:"summary"`
	ProjectKey  string         `json:"project_key"`
	JiraURL     string         `json:"jira_url,omitempty"`
	BodyFormat  string         `json:"body_format,omitempty"`
	JiraUpdated *time.Time     `json:"jira_updated,omitempty"`
	Attachments []AttachmentRef `json:"attachments,omitempty"`
	SavedAt     time.Time      `json:"saved_at"`
}

// AttachmentRef records attachment metadata and optional local path.
type AttachmentRef struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	MIMEType  string `json:"mimeType,omitempty"`
	Content   string `json:"content,omitempty"`
	Size      int64  `json:"size,omitempty"`
	LocalPath string `json:"local_path,omitempty"` // relative to issue dir when set
}

// Writer writes per-issue artifacts under saved/{projectKey}/{issueKey}/.
type Writer struct {
	basePath string
	log      logging.Sugar
}

// NewWriter creates a storage writer rooted at basePath (usually config storage).
func NewWriter(basePath string, log logging.Sugar) *Writer {
	return &Writer{basePath: basePath, log: log}
}

// IssueDir returns saved/{project}/{issueKey}/ and ensures it exists.
func (w *Writer) IssueDir(projectKey, issueKey string) (string, error) {
	dir := filepath.Join(w.basePath, strings.ToUpper(projectKey), strings.ToUpper(issueKey))
	if err := os.MkdirAll(filepath.Join(dir, "assets", "attachments"), 0o755); err != nil {
		return "", fmt.Errorf("create issue dir: %w", err)
	}
	return dir, nil
}

// SaveFieldsJSON writes fields.json (raw Jira fields object).
func (w *Writer) SaveFieldsJSON(dir string, fields json.RawMessage) (string, error) {
	path := filepath.Join(dir, "fields.json")
	if err := os.WriteFile(path, fields, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// SaveMarkdown writes content.md.
func (w *Writer) SaveMarkdown(dir, markdown string) (string, error) {
	path := filepath.Join(dir, "content.md")
	if err := os.WriteFile(path, []byte(markdown), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// SaveMetadata writes metadata.json.
func (w *Writer) SaveMetadata(dir string, meta Metadata) (string, error) {
	meta.SavedAt = time.Now().UTC()
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "metadata.json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
