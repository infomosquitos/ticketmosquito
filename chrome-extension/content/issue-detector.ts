/** Parse Jira URL (and optional DOM) into project/issue context for the Issue tab. */
export function detectCurrentIssue(): {
  projectKey: string;
  projectName: string;
  projectURL: string;
  issueKey: string;
  summary: string;
  host: string;
  tabUrl: string;
} {
  const url = window.location.href;
  const hostname = window.location.hostname;
  const parsed = new URL(url);

  let issueKey = '';
  let projectKey = '';

  const browseMatch = url.match(/\/browse\/([A-Z][A-Z0-9]+-\d+)/i);
  if (browseMatch) issueKey = browseMatch[1].toUpperCase();

  const jiraBrowse = url.match(/\/jira\/browse\/([A-Z][A-Z0-9]+-\d+)/i);
  if (jiraBrowse) issueKey = jiraBrowse[1].toUpperCase();

  const projMatch =
    url.match(/\/projects\/([^/?#]+)/i) ||
    url.match(/\/jira(?:\/software)?\/c?\/projects\/([^/?#]+)/i);
  if (projMatch) projectKey = decodeURIComponent(projMatch[1]).toUpperCase();

  const selected = parsed.searchParams.get('selectedIssue');
  if (selected) issueKey = selected.toUpperCase();

  if (issueKey && !projectKey) {
    const m = issueKey.match(/^([A-Z][A-Z0-9]+)-\d+$/);
    if (m) projectKey = m[1];
  }

  let projectURL = url;
  if (projectKey) {
    projectURL = `${parsed.protocol}//${hostname}/projects/${encodeURIComponent(projectKey)}`;
  }

  let summary = document.title.trim();
  summary = summary.replace(/\s*[-–|]\s*Jira.*$/i, '').trim();

  return {
    projectKey,
    projectName: projectKey,
    projectURL,
    issueKey,
    summary,
    host: hostname,
    tabUrl: url,
  };
}
