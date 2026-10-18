CREATE VIRTUAL TABLE issues_fts USING fts5(
    issue_id UNINDEXED,
    summary,
    content,
    tokenize='porter'
);

CREATE TRIGGER issues_fts_insert AFTER INSERT ON issues BEGIN
    INSERT INTO issues_fts(issue_id, summary, content)
    VALUES (new.id, new.summary, coalesce(new.content, ''));
END;

CREATE TRIGGER issues_fts_delete AFTER DELETE ON issues BEGIN
    DELETE FROM issues_fts WHERE issue_id = old.id;
END;

CREATE TRIGGER issues_fts_update AFTER UPDATE OF summary, content ON issues BEGIN
    DELETE FROM issues_fts WHERE issue_id = old.id;
    INSERT INTO issues_fts(issue_id, summary, content)
    VALUES (new.id, new.summary, coalesce(new.content, ''));
END;

INSERT INTO issues_fts(issue_id, summary, content)
SELECT id, summary, coalesce(content, '') FROM issues;
