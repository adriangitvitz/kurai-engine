package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAPIKey_Deterministic(t *testing.T) {
	h1 := HashAPIKeyHex("test-key-12345")
	h2 := HashAPIKeyHex("test-key-12345")
	assert.Equal(t, h1, h2)
}

func TestHashAPIKey_DifferentKeys(t *testing.T) {
	h1 := HashAPIKeyHex("key-one")
	h2 := HashAPIKeyHex("key-two")
	assert.NotEqual(t, h1, h2)
}

func TestHashAPIKey_OutputFormat(t *testing.T) {
	h := HashAPIKeyHex("any-key")
	assert.Len(t, h, 64, "hash should be 64 hex chars")
}

func TestHashAPIKey_CrossValidation(t *testing.T) {
	h := HashAPIKeyHex("kurai-test-vector-v1")
	require.Len(t, h, 64)
	t.Logf("Cross-validation vector: HashAPIKeyHex('kurai-test-vector-v1') = %s", h)
}

func TestVerifyAPIKey_Valid(t *testing.T) {
	hash := HashAPIKey("my-secret-key")
	assert.True(t, VerifyAPIKey("my-secret-key", hash))
}

func TestVerifyAPIKey_Wrong(t *testing.T) {
	hash := HashAPIKey("my-secret-key")
	assert.False(t, VerifyAPIKey("wrong-key", hash))
}

func TestDomainSeparator(t *testing.T) {
	expected := [32]byte{}
	copy(expected[:], "kurai_api_key_hash_v1")
	assert.Equal(t, expected, domainSeparator)
}
