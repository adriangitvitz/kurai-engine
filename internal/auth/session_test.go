package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAndValidateSession(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)

	token, expiry := sm.CreateSession("alice")
	require.NotEmpty(t, token)
	assert.True(t, expiry.After(time.Now()))

	userID, err := sm.ValidateSession(token)
	require.NoError(t, err)
	assert.Equal(t, "alice", userID)
}

func TestSessionExpiry(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 0, false)

	token, _ := sm.CreateSession("alice")

	time.Sleep(time.Second)
	_, err := sm.ValidateSession(token)
	assert.ErrorIs(t, err, ErrExpiredSession)
}

func TestSessionTampered(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)

	token, _ := sm.CreateSession("alice")

	tampered := []byte(token)
	if tampered[10] == 'a' {
		tampered[10] = 'b'
	} else {
		tampered[10] = 'a'
	}

	_, err := sm.ValidateSession(string(tampered))
	assert.Error(t, err)
}

func TestSessionDifferentSecret(t *testing.T) {
	sm1 := NewSessionManager([]byte("secret-one-32-bytes-minimum!!!!"), 8*time.Hour, false)
	sm2 := NewSessionManager([]byte("secret-two-32-bytes-minimum!!!!"), 8*time.Hour, false)

	token, _ := sm1.CreateSession("alice")

	_, err := sm2.ValidateSession(token)
	assert.Error(t, err)
}

func TestSessionNonceUniqueness(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)

	token1, _ := sm.CreateSession("alice")
	token2, _ := sm.CreateSession("alice")

	assert.NotEqual(t, token1, token2, "two tokens for same user should differ (nonce)")
}

func TestSessionInvalidBase64(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)

	_, err := sm.ValidateSession("not-valid-base64!!!")
	assert.Error(t, err)
}

func TestSessionTooShort(t *testing.T) {
	sm := NewSessionManager([]byte("test-secret-32-bytes-minimum!!!"), 8*time.Hour, false)

	_, err := sm.ValidateSession("dG9vc2hvcnQ")
	assert.Error(t, err)
}
