package scraper

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/infomosquitos/ticketmosquito/internal/contentmd"
	"github.com/infomosquitos/ticketmosquito/internal/jira"
	"github.com/infomosquitos/ticketmosquito/internal/session"
	"github.com/infomosquitos/ticketmosquito/internal/storage"
	"github.com/infomosquitos/ticketmosquito/internal/store"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// Scraper crawls Jira projects/issues via REST (API-only; no browser fallback).
type Scraper struct {
	db       store.Store
	sessions *session.Store
	encKey   string
	writer   *storage.Writer
	force    bool
	log      logging.Sugar
}

// New creates a scraper.
func New(db store.Store, sessions *session.Store, encKey string, writer *storage.Writer, log logging.Sugar) *Scraper {
	return &Scraper{db: db, sessions: sessions, encKey: encKey, writer: writer, log: log}
}

// SetForce enables full refresh (ignore updated skip).
func (s *Scraper) SetForce(force bool) { s.force = force }

// Stats summarizes a crawl run.
type Stats struct {
	Discovered int
	Fetched    int
	Skipped    int
	Failed     int
}

// CrawlProject crawls all issues in a project (URL or bare key).
func (s *Scraper) CrawlProject(ctx context.Context, projectRef string) (*Stats, error) {
	projectKey, _, err := jira.ParseProjectRef(projectRef)
	if err != nil {
		return nil, err
	}
	sess, siteURL, err := s.resolveSession(projectRef)
	if err != nil {
		return nil, err
	}
	client, err := newJiraClient(sess, s.log)
	if err != nil {
		return nil, err
	}

	projectURL := jira.BrowseURL(siteURL, projectKey)
	projectID, err := s.db.UpsertProject(ctx, projectKey, projectKey, projectURL)
	if err != nil {
		return nil, err
	}

	refs, err := client.SearchProjectIssues(ctx, projectKey)
	if err != nil {
		return nil, fmt.Errorf("discover issues: %w", err)
	}
	stats := &Stats{Discovered: len(refs)}
	s.log.Infow("discovered issues", "project", projectKey, "count", len(refs), "force", s.force)

	for _, ref := range refs {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		skip, err := s.shouldSkip(ctx, projectKey, ref)
		if err != nil {
			return stats, err
		}
		if skip {
			stats.Skipped++
			continue
		}
		if err := s.fetchAndStore(ctx, client, projectID, projectKey, siteURL, ref.Key); err != nil {
			stats.Failed++
			s.log.Errorw("issue crawl failed", "key", ref.Key, "error", err)
			if jira.IsUnauthorized(err) {
				return stats, err
			}
			continue
		}
		stats.Fetched++
	}

	_ = s.db.TouchProjectCrawl(ctx, projectKey, len(refs), time.Now().UTC())
	return stats, nil
}

// CrawlIssue refreshes a single issue (always overwrites text; no updated skip).
func (s *Scraper) CrawlIssue(ctx context.Context, issueKey string) error {
	projectKey, key, err := jira.ParseProjectRef(issueKey)
	if err != nil {
		return err
	}
	if key == "" {
		return fmt.Errorf("expected issue key like PROJ-123, got %q", issueKey)
	}
	sess, siteURL, err := s.resolveSession(issueKey)
	if err != nil {
		return err
	}
	client, err := newJiraClient(sess, s.log)
	if err != nil {
		return err
	}
	projectURL := jira.BrowseURL(siteURL, projectKey)
	projectID, err := s.db.UpsertProject(ctx, projectKey, projectKey, projectURL)
	if err != nil {
		return err
	}
	return s.fetchAndStore(ctx, client, projectID, projectKey, siteURL, key)
}

func (s *Scraper) shouldSkip(ctx context.Context, projectKey string, ref jira.IssueRef) (bool, error) {
	if s.force || ref.Updated.IsZero() {
		return false, nil
	}
	existing, err := s.db.GetIssueByJiraID(ctx, projectKey, ref.ID)
	if errors.Is(err, store.ErrNotFound) {
		existing, err = s.db.GetIssueByKey(ctx, ref.Key)
	}
	if errors.Is(err, store.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if existing.JiraUpdated == nil {
		return false, nil
	}
	if !ref.Updated.After(*existing.JiraUpdated) {
		return true, nil
	}
	return false, nil
}

func (s *Scraper) fetchAndStore(ctx context.Context, client *jira.Client, projectID uuid.UUID, projectKey, siteURL, issueKey string) error {
	detail, err := client.GetIssue(ctx, issueKey)
	if err != nil {
		return err
	}
	comments, err := client.ListComments(ctx, issueKey)
	if err != nil {
		return err
	}

	in := contentmd.IssueContentInput{
		Description: contentmd.BodyInput{
			ADF:  detail.DescriptionADF,
			Wiki: detail.DescriptionWiki,
			HTML: detail.DescriptionHTML,
		},
	}
	for _, c := range comments {
		in.Comments = append(in.Comments, contentmd.CommentInput{
			Author:  c.Author,
			Created: c.Created,
			Body: contentmd.BodyInput{
				ADF:  c.BodyADF,
				Wiki: c.BodyWiki,
				HTML: c.BodyHTML,
			},
		})
	}
	rendered, err := contentmd.RenderIssue(in)
	if err != nil {
		return fmt.Errorf("contentmd: %w", err)
	}

	dir, err := s.writer.IssueDir(projectKey, detail.Key)
	if err != nil {
		return err
	}
	if _, err := s.writer.SaveFieldsJSON(dir, detail.FieldsJSON); err != nil {
		return err
	}
	contentPath, err := s.writer.SaveMarkdown(dir, rendered.Markdown)
	if err != nil {
		return err
	}
	attRefs := s.downloadAttachments(client, dir, detail.Attachments)
	var jiraUpdated *time.Time
	if !detail.Updated.IsZero() {
		u := detail.Updated
		jiraUpdated = &u
	}
	browse := strings.TrimRight(jira.BaseURL(siteURL), "/") + "/browse/" + detail.Key
	metaPath, err := s.writer.SaveMetadata(dir, storage.Metadata{
		Key: detail.Key, JiraID: detail.ID, Summary: detail.Summary, ProjectKey: projectKey,
		JiraURL: browse, BodyFormat: string(rendered.BodyFormat),
		JiraUpdated: jiraUpdated, Attachments: attRefs,
	})
	if err != nil {
		return err
	}

	iss := &store.Issue{
		ProjectID:    projectID,
		JiraID:       detail.ID,
		Key:          detail.Key,
		Summary:      detail.Summary,
		Content:      rendered.Markdown,
		FieldsJSON:   string(detail.FieldsJSON),
		FileDir:      dir,
		ContentPath:  contentPath,
		MetadataPath: metaPath,
		BodyFormat:   string(rendered.BodyFormat),
		JiraUpdated:  jiraUpdated,
	}
	if err := s.db.UpsertIssue(ctx, iss); err != nil {
		return err
	}
	s.log.Infow("issue saved", "key", detail.Key, "format", rendered.BodyFormat, "comments", len(comments))
	return nil
}

func (s *Scraper) resolveSession(ref string) (*session.Session, string, error) {
	if s.sessions == nil {
		return nil, "", fmt.Errorf("session store not configured")
	}
	if s.encKey == "" {
		return nil, "", fmt.Errorf("encryption key not configured")
	}

	if strings.Contains(ref, "://") {
		sess, err := s.sessions.GetForURL(s.encKey, ref)
		if err == nil {
			return ensureFlavor(sess), sess.SiteURL, nil
		}
		if !errors.Is(err, session.ErrNoSessionForHost) {
			return nil, "", err
		}
	}

	sess, err := s.sessions.Load(s.encKey)
	if err != nil {
		return nil, "", fmt.Errorf("load session: %w (capture a Jira session first)", err)
	}
	return ensureFlavor(sess), sess.SiteURL, nil
}

func ensureFlavor(sess *session.Session) *session.Session {
	if sess.Flavor == "" {
		if strings.Contains(strings.ToLower(sess.SiteURL), "atlassian.net") {
			sess.Flavor = session.FlavorCloud
		} else {
			sess.Flavor = session.FlavorServer
		}
	}
	return sess
}

func newJiraClient(sess *session.Session, log logging.Sugar) (*jira.Client, error) {
	return jira.NewClient(jira.ClientConfig{
		APIBase: sess.APIBase(),
		Server:  sess.Flavor == session.FlavorServer,
		Headers: sess.AsHeaders(),
	}, log)
}

// downloadAttachments fetches binaries best-effort; failures are logged and do not fail the issue.
func (s *Scraper) downloadAttachments(client *jira.Client, issueDir string, attachments []jira.AttachmentMeta) []storage.AttachmentRef {
	refs := make([]storage.AttachmentRef, 0, len(attachments))
	if len(attachments) == 0 {
		return refs
	}

	dl := storage.NewAssetDownloader(s.log)
	dl.SetForce(s.force)
	dl.SetAuthHeaders(client.AuthHeaders())

	attDir := storage.AttachmentsDir(issueDir)
	for _, a := range attachments {
		ref := storage.AttachmentRef{
			ID: a.ID, Filename: a.Filename, MIMEType: a.MIMEType, Content: a.Content, Size: a.Size,
		}
		downloadURL := storage.ResolveAttachmentURL(a.Content, client.APIBase(), a.ID)
		if downloadURL == "" {
			s.log.Warnw("attachment missing download URL", "id", a.ID, "filename", a.Filename)
			refs = append(refs, ref)
			continue
		}
		dest := filepath.Join(attDir, storage.StableAttachmentName(a.ID, a.Filename))
		if err := dl.DownloadAs(dest, downloadURL); err != nil {
			s.log.Warnw("attachment download failed", "id", a.ID, "filename", a.Filename, "error", err)
			refs = append(refs, ref)
			continue
		}
		ref.LocalPath = storage.RelativeToIssue(issueDir, dest)
		refs = append(refs, ref)
	}
	return refs
}
