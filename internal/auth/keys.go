package auth

import (
	"crypto/subtle"
	"encoding/hex"

	"github.com/zeebo/blake3"
)

var domainSeparator = func() [32]byte {
	var key [32]byte
	copy(key[:], "kurai_api_key_hash_v1")
	return key
}()

func HashAPIKey(rawKey string) [32]byte {
	h, _ := blake3.NewKeyed(domainSeparator[:])
	h.Write([]byte(rawKey))
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func HashAPIKeyHex(rawKey string) string {
	h := HashAPIKey(rawKey)
	return hex.EncodeToString(h[:])
}

func VerifyAPIKey(rawKey string, storedHash [32]byte) bool {
	computed := HashAPIKey(rawKey)
	return subtle.ConstantTimeCompare(computed[:], storedHash[:]) == 1
}
