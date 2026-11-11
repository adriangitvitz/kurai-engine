package bookmarks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS bookmarks (
	user_id      TEXT NOT NULL,
	url          TEXT NOT NULL,
	title        TEXT NOT NULL DEFAULT '',
	snippet      TEXT NOT NULL DEFAULT '',
	domain       TEXT NOT NULL DEFAULT '',
	content_type TEXT NOT NULL DEFAULT '',
	meta         TEXT NOT NULL DEFAULT '',
	score        REAL NOT NULL DEFAULT 0,
	query        TEXT,
	saved_at     TIMESTAMP NOT NULL,
	PRIMARY KEY (user_id, url)
);
CREATE INDEX IF NOT EXISTS idx_bookmarks_user_saved
	ON bookmarks (user_id, saved_at DESC);
`

type SQLiteStore struct {
	db  *sql.DB
	now func() time.Time
}

func OpenSQLite(path string) (*SQLiteStore, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating bookmark data directory: %w", err)
		}
	}

	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening bookmark database: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to bookmark database: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying bookmark schema: %w", err)
	}
	return &SQLiteStore{db: db, now: time.Now}, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

var ErrNoURL = errors.New("bookmark url is required")

func (s *SQLiteStore) Save(ctx context.Context, userID string, b Bookmark) (Bookmark, error) {
	if b.URL == "" {
		return Bookmark{}, ErrNoURL
	}
	if b.SavedAt.IsZero() {
		b.SavedAt = s.now().UTC()
	}

	var query any
	if b.Query != "" {
		query = b.Query
	}

	const stmt = `
INSERT INTO bookmarks (user_id, url, title, snippet, domain, content_type, meta, score, query, saved_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, url) DO UPDATE SET
	title        = excluded.title,
	snippet      = excluded.snippet,
	domain       = excluded.domain,
	content_type = excluded.content_type,
	meta         = excluded.meta,
	score        = excluded.score,
	query        = COALESCE(bookmarks.query, excluded.query)
`
	if _, err := s.db.ExecContext(ctx, stmt,
		userID, b.URL, b.Title, b.Snippet, b.Domain, b.ContentType, b.Meta, b.Score, query, b.SavedAt,
	); err != nil {
		return Bookmark{}, fmt.Errorf("saving bookmark: %w", err)
	}

	return s.get(ctx, userID, b.URL)
}

func (s *SQLiteStore) get(ctx context.Context, userID, url string) (Bookmark, error) {
	const stmt = `
SELECT url, title, snippet, domain, content_type, meta, score, COALESCE(query, ''), saved_at
FROM bookmarks WHERE user_id = ? AND url = ?`
	var b Bookmark
	err := s.db.QueryRowContext(ctx, stmt, userID, url).Scan(
		&b.URL, &b.Title, &b.Snippet, &b.Domain, &b.ContentType, &b.Meta, &b.Score, &b.Query, &b.SavedAt,
	)
	if err != nil {
		return Bookmark{}, fmt.Errorf("reading bookmark: %w", err)
	}
	b.SavedAt = b.SavedAt.UTC()
	return b, nil
}

func (s *SQLiteStore) List(ctx context.Context, userID string) ([]Bookmark, error) {
	const stmt = `
SELECT url, title, snippet, domain, content_type, meta, score, COALESCE(query, ''), saved_at
FROM bookmarks WHERE user_id = ?
ORDER BY saved_at DESC, url ASC`
	rows, err := s.db.QueryContext(ctx, stmt, userID)
	if err != nil {
		return nil, fmt.Errorf("listing bookmarks: %w", err)
	}
	defer rows.Close()

	out := []Bookmark{}
	for rows.Next() {
		var b Bookmark
		if err := rows.Scan(&b.URL, &b.Title, &b.Snippet, &b.Domain, &b.ContentType,
			&b.Meta, &b.Score, &b.Query, &b.SavedAt); err != nil {
			return nil, fmt.Errorf("scanning bookmark: %w", err)
		}
		b.SavedAt = b.SavedAt.UTC()
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ListGrouped(ctx context.Context, userID string) ([]Group, error) {
	all, err := s.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	groups := []Group{}
	index := map[string]int{}
	for _, b := range all {
		i, seen := index[b.Query]
		if !seen {
			groups = append(groups, Group{Query: b.Query})
			i = len(groups) - 1
			index[b.Query] = i
		}
		groups[i].Bookmarks = append(groups[i].Bookmarks, b)
	}
	return groups, nil
}

func (s *SQLiteStore) Delete(ctx context.Context, userID, url string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM bookmarks WHERE user_id = ? AND url = ?`, userID, url)
	if err != nil {
		return fmt.Errorf("deleting bookmark: %w", err)
	}
	return nil
}
