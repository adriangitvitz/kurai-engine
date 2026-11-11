package bookmarks

import (
	"context"
	"time"
)

type Bookmark struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Snippet     string    `json:"snippet,omitempty"`
	Domain      string    `json:"domain,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
	Meta        string    `json:"meta,omitempty"`
	Score       float64   `json:"score,omitempty"`
	Query       string    `json:"query,omitempty"`
	SavedAt     time.Time `json:"saved_at"`
}

type Group struct {
	Query     string     `json:"query,omitempty"`
	Bookmarks []Bookmark `json:"bookmarks"`
}

type Store interface {
	Save(ctx context.Context, userID string, b Bookmark) (Bookmark, error)
	List(ctx context.Context, userID string) ([]Bookmark, error)
	ListGrouped(ctx context.Context, userID string) ([]Group, error)
	Delete(ctx context.Context, userID, url string) error
	Close() error
}
