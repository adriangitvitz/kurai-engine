package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"time"

	"golang.org/x/crypto/hkdf"
)

var (
	ErrInvalidSession = errors.New("invalid session token")
	ErrExpiredSession = errors.New("session expired")
)

type SessionManager struct {
	signingKey [32]byte
	ttl        time.Duration
	cookieName string
	secure     bool
}

func NewSessionManager(serverSecret []byte, ttl time.Duration, secure bool) *SessionManager {
	sm := &SessionManager{
		ttl:        ttl,
		cookieName: "kurai_session",
		secure:     secure,
	}

	reader := hkdf.New(sha256.New, serverSecret, nil, []byte("KuraiSession-v1-HMAC"))
	if _, err := io.ReadFull(reader, sm.signingKey[:]); err != nil {
		panic("hkdf: " + err.Error())
	}

	return sm
}

func (sm *SessionManager) CreateSession(userID string) (string, time.Time) {
	expiry := time.Now().Add(sm.ttl)
	expiryUnix := uint64(expiry.Unix())

	userIDBytes := []byte(userID)
	payloadLen := 2 + len(userIDBytes) + 8 + 16
	payload := make([]byte, payloadLen)

	offset := 0
	binary.BigEndian.PutUint16(payload[offset:], uint16(len(userIDBytes)))
	offset += 2
	copy(payload[offset:], userIDBytes)
	offset += len(userIDBytes)
	binary.BigEndian.PutUint64(payload[offset:], expiryUnix)
	offset += 8
	if _, err := rand.Read(payload[offset : offset+16]); err != nil {
		panic("rand: " + err.Error())
	}

	mac := hmac.New(sha256.New, sm.signingKey[:])
	mac.Write(payload)
	sig := mac.Sum(nil)

	token := make([]byte, len(payload)+len(sig))
	copy(token, payload)
	copy(token[len(payload):], sig)

	return base64.RawURLEncoding.EncodeToString(token), expiry
}

func (sm *SessionManager) ValidateSession(token string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", ErrInvalidSession
	}

	if len(raw) < 59 {
		return "", ErrInvalidSession
	}

	userIDLen := int(binary.BigEndian.Uint16(raw[:2]))
	if userIDLen == 0 || 2+userIDLen+8+16+32 != len(raw) {
		return "", ErrInvalidSession
	}

	payloadLen := 2 + userIDLen + 8 + 16
	payload := raw[:payloadLen]
	sig := raw[payloadLen:]

	if len(sig) != 32 {
		return "", ErrInvalidSession
	}

	mac := hmac.New(sha256.New, sm.signingKey[:])
	mac.Write(payload)
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return "", ErrInvalidSession
	}

	offset := 2
	userID := string(payload[offset : offset+userIDLen])
	offset += userIDLen
	expiryUnix := binary.BigEndian.Uint64(payload[offset:])

	if time.Now().Unix() > int64(expiryUnix) {
		return "", ErrExpiredSession
	}

	return userID, nil
}

func (sm *SessionManager) CookieName() string {
	return sm.cookieName
}

func (sm *SessionManager) TTL() time.Duration {
	return sm.ttl
}

func (sm *SessionManager) Secure() bool {
	return sm.secure
}
