package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/adriangitvitz/kurai/internal/apierror"
	"github.com/adriangitvitz/kurai/internal/auth"
	"github.com/adriangitvitz/kurai/internal/bookmarks"
)

type bookmarkRequest struct {
	URL         string  `json:"url"`
	Title       string  `json:"title"`
	Snippet     string  `json:"snippet"`
	Domain      string  `json:"domain"`
	ContentType string  `json:"content_type"`
	Meta        string  `json:"meta"`
	Score       float64 `json:"score"`
	Query       string  `json:"query"`
}

type BookmarksResponse struct {
	Bookmarks []bookmarks.Bookmark `json:"bookmarks"`
	Envelope  Envelope             `json:"envelope,omitempty"`
}

type GroupedBookmarksResponse struct {
	Groups   []bookmarks.Group `json:"groups"`
	Envelope Envelope          `json:"envelope,omitempty"`
}

func (s *Server) handleBookmarks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListBookmarks(w, r)
	case http.MethodPost:
		s.handleSaveBookmark(w, r)
	case http.MethodDelete:
		s.handleDeleteBookmark(w, r)
	default:
		apierror.Write(w, http.StatusMethodNotAllowed, apierror.CodeMethodNotAllowed,
			"method not allowed on /bookmarks", RequestIDFromContext(r.Context()))
	}
}

func (s *Server) handleListBookmarks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	if r.URL.Query().Get("group") == "query" {
		groups, err := s.bookmarks.ListGrouped(ctx, userID)
		if err != nil {
			slog.Error("bookmarks: grouped list failed", "error", err)
			apierror.Write(w, http.StatusInternalServerError, apierror.CodeInternalError,
				"failed to read the board", RequestIDFromContext(ctx))
			return
		}
		s.writeJSON(w, http.StatusOK, &GroupedBookmarksResponse{
			Groups:   groups,
			Envelope: Envelope{RequestID: RequestIDFromContext(ctx)},
		})
		return
	}

	list, err := s.bookmarks.List(ctx, userID)
	if err != nil {
		slog.Error("bookmarks: list failed", "error", err)
		apierror.Write(w, http.StatusInternalServerError, apierror.CodeInternalError,
			"failed to read the board", RequestIDFromContext(ctx))
		return
	}
	s.writeJSON(w, http.StatusOK, &BookmarksResponse{
		Bookmarks: list,
		Envelope:  Envelope{RequestID: RequestIDFromContext(ctx)},
	})
}

func (s *Server) handleSaveBookmark(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req bookmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Write(w, http.StatusBadRequest, apierror.CodeInvalidRequest,
			"invalid JSON body", RequestIDFromContext(ctx))
		return
	}

	saved, err := s.bookmarks.Save(ctx, auth.UserIDFromContext(ctx), bookmarks.Bookmark{
		URL:         req.URL,
		Title:       req.Title,
		Snippet:     req.Snippet,
		Domain:      req.Domain,
		ContentType: req.ContentType,
		Meta:        req.Meta,
		Score:       req.Score,
		Query:       req.Query,
	})
	if errors.Is(err, bookmarks.ErrNoURL) {
		apierror.Write(w, http.StatusBadRequest, apierror.CodeInvalidRequest,
			"url is required", RequestIDFromContext(ctx))
		return
	}
	if err != nil {
		slog.Error("bookmarks: save failed", "error", err)
		apierror.Write(w, http.StatusInternalServerError, apierror.CodeInternalError,
			"failed to save the bookmark", RequestIDFromContext(ctx))
		return
	}
	s.writeJSON(w, http.StatusOK, saved)
}

func (s *Server) handleDeleteBookmark(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	url := r.URL.Query().Get("url")
	if url == "" {
		apierror.Write(w, http.StatusBadRequest, apierror.CodeInvalidRequest,
			"url query parameter is required", RequestIDFromContext(ctx))
		return
	}

	if err := s.bookmarks.Delete(ctx, auth.UserIDFromContext(ctx), url); err != nil {
		slog.Error("bookmarks: delete failed", "error", err)
		apierror.Write(w, http.StatusInternalServerError, apierror.CodeInternalError,
			"failed to delete the bookmark", RequestIDFromContext(ctx))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
