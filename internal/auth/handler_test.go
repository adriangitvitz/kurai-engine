package auth

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.in/yaml.v3"
)

func setupTestAuthHandler(t *testing.T) (*AuthHandler, string) {
	t.Helper()

	rawKey := "test-api-key-12345"
	hash := HashAPIKey(rawKey)
	hashHex := hex.EncodeToString(hash[:])

	usersData := map[string]any{
		"users": []map[string]any{
			{
				"id":       "testuser",
				"name":     "Test User",
				"key_hash": hashHex,
				"enabled":  true,
			},
		},
	}

	tmpFile, err := os.CreateTemp("", "users-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	data, err := yaml.Marshal(usersData)
	require.NoError(t, err)
	_, err = tmpFile.Write(data)
	require.NoError(t, err)
	tmpFile.Close()

	store := NewUserStore()
	require.NoError(t, store.LoadFromFile(tmpFile.Name()))

	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)
	handler := NewAuthHandler(store, sm, 5)

	return handler, rawKey
}

func TestHandleAuth_ValidKey(t *testing.T) {
	handler, rawKey := setupTestAuthHandler(t)

	body := `{"key":"` + rawKey + `"}`
	req := httptest.NewRequest(http.MethodPost, "/auth", strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	cookies := rec.Result().Cookies()
	require.NotEmpty(t, cookies)
	assert.Equal(t, "kurai_session", cookies[0].Name)

	var resp AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "testuser", resp.UserID)
}

func TestHandleAuth_InvalidKey(t *testing.T) {
	handler, _ := setupTestAuthHandler(t)

	body := `{"key":"wrong-key"}`
	req := httptest.NewRequest(http.MethodPost, "/auth", strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Empty(t, rec.Result().Cookies())
}

func TestHandleAuth_WrongMethod(t *testing.T) {
	handler, _ := setupTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestHandleAuth_MissingBody(t *testing.T) {
	handler, _ := setupTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/auth", strings.NewReader(""))
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleAuth_RateLimit(t *testing.T) {
	handler, _ := setupTestAuthHandler(t)

	for range 5 {
		body := `{"key":"wrong-key"}`
		req := httptest.NewRequest(http.MethodPost, "/auth", strings.NewReader(body))
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	body := `{"key":"wrong-key"}`
	req := httptest.NewRequest(http.MethodPost, "/auth", strings.NewReader(body))
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}
