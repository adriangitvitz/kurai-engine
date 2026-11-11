package auth

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/apierror"
)

func userStoreWithKey(t *testing.T, userID, rawKey string) *UserStore {
	t.Helper()
	hash := HashAPIKey(rawKey)
	store := NewUserStore()
	store.users[userID] = &User{
		ID:      userID,
		Name:    userID,
		KeyHash: hex.EncodeToString(hash[:]),
		Enabled: true,
	}
	store.hashIdx[hash] = userID
	return store
}

func TestRequireAuth_BearerToken_ValidKeyAuthorizes(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)
	store := userStoreWithKey(t, "alice", "alice-secret-key")
	limiter := NewRateLimiter(60)

	var gotUserID string
	handler := RequireAuth(sm, store, limiter, true)(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	req.Header.Set("Authorization", "Bearer alice-secret-key")
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "alice", gotUserID)
}

func TestRequireAuth_BearerToken_InvalidKeyRejects(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)
	store := userStoreWithKey(t, "alice", "alice-secret-key")
	limiter := NewRateLimiter(60)

	handler := RequireAuth(sm, store, limiter, true)(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Header().Get("WWW-Authenticate"), "Bearer")

	var env apierror.Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, apierror.CodeUnauthorized, env.Error.Code)
}

func TestRequireAuth_BearerToken_TakesPrecedenceOverCookie(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)
	store := userStoreWithKey(t, "alice", "alice-secret-key")
	limiter := NewRateLimiter(60)

	cookieToken, _ := sm.CreateSession("bob")

	var gotUserID string
	handler := RequireAuth(sm, store, limiter, true)(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	req.Header.Set("Authorization", "Bearer alice-secret-key")
	req.AddCookie(&http.Cookie{Name: sm.CookieName(), Value: cookieToken})
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "alice", gotUserID, "Bearer must win when both are present")
}

func TestRequireAuth_BearerToken_FallsBackToCookieWhenHeaderAbsent(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)
	store := userStoreWithKey(t, "alice", "alice-secret-key")
	limiter := NewRateLimiter(60)

	cookieToken, _ := sm.CreateSession("bob")

	var gotUserID string
	handler := RequireAuth(sm, store, limiter, true)(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	req.AddCookie(&http.Cookie{Name: sm.CookieName(), Value: cookieToken})
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "bob", gotUserID)
}

func TestRequireAuth_BearerToken_MalformedHeader(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)
	store := userStoreWithKey(t, "alice", "alice-secret-key")
	limiter := NewRateLimiter(60)

	handler := RequireAuth(sm, store, limiter, true)(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not be called")
	})

	cases := []string{
		"Basic alice-secret-key",
		"Bearer",
		"Bearer ",
		"alice-secret-key",
	}
	for _, header := range cases {
		req := httptest.NewRequest(http.MethodGet, "/search", nil)
		req.Header.Set("Authorization", header)
		rec := httptest.NewRecorder()
		handler(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "header=%q", header)
	}
}

func TestRequireAuth_BearerToken_RateLimitsRepeatedAttempts(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)
	store := userStoreWithKey(t, "alice", "alice-secret-key")
	limiter := NewRateLimiter(3)

	handler := RequireAuth(sm, store, limiter, true)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	makeReq := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/search", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		req.Header.Set("Authorization", "Bearer wrong-key")
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}

	for i := 0; i < 3; i++ {
		assert.Equal(t, http.StatusUnauthorized, makeReq().Code, "attempt %d", i+1)
	}
	rec := makeReq()
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "4th attempt must hit rate limit")

	var env apierror.Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, apierror.CodeRateLimited, env.Error.Code)
}

func TestRequireAuth_BearerToken_RateLimitOnlyCountsFailures(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)
	store := userStoreWithKey(t, "alice", "alice-secret-key")
	limiter := NewRateLimiter(2)

	handler := RequireAuth(sm, store, limiter, true)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/search", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		req.Header.Set("Authorization", "Bearer alice-secret-key")
		rec := httptest.NewRecorder()
		handler(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "successful request %d must not consume the failure budget", i+1)
	}
}

func TestRequireAuth_BearerToken_DisabledAuthInjectsAnonymous(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)
	store := userStoreWithKey(t, "alice", "alice-secret-key")
	limiter := NewRateLimiter(60)

	var gotUserID string
	handler := RequireAuth(sm, store, limiter, false)(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "anonymous", gotUserID)
}
