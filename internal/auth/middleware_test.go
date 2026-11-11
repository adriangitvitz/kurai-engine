package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRequireAuth_NoCookie(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)

	handler := RequireAuth(sm, nil, nil, true)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAuth_InvalidCookie(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)

	handler := RequireAuth(sm, nil, nil, true)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	req.AddCookie(&http.Cookie{Name: sm.CookieName(), Value: "garbage"})
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAuth_ExpiredCookie(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 0, false)

	token, _ := sm.CreateSession("alice")
	time.Sleep(time.Second)

	handler := RequireAuth(sm, nil, nil, true)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	req.AddCookie(&http.Cookie{Name: sm.CookieName(), Value: token})
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAuth_ValidCookie(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)

	token, _ := sm.CreateSession("alice")

	var gotUserID string
	handler := RequireAuth(sm, nil, nil, true)(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	req.AddCookie(&http.Cookie{Name: sm.CookieName(), Value: token})
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "alice", gotUserID)
}

func TestRequireAuth_Disabled(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)

	var gotUserID string
	handler := RequireAuth(sm, nil, nil, false)(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "anonymous", gotUserID)
}
