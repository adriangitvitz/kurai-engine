package apierror

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrite_EncodesEnvelopeAndStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	Write(rec, http.StatusBadRequest, CodeMissingQuery, "query parameter 'q' is required", "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var env Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, CodeMissingQuery, env.Error.Code)
	assert.Equal(t, "query parameter 'q' is required", env.Error.Message)
	assert.Empty(t, env.RequestID)
	assert.Nil(t, env.Error.Details)
}

func TestWrite_IncludesRequestIDWhenSet(t *testing.T) {
	rec := httptest.NewRecorder()
	Write(rec, http.StatusInternalServerError, CodeInternalError, "boom", "req_abc123")

	var env Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, "req_abc123", env.RequestID)
}

func TestWriteWithDetails_IncludesDetails(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteWithDetails(rec, http.StatusBadRequest, CodeInvalidFilter, "bad filter",
		map[string]any{"field": "date_from", "value": "not-a-date"}, "")

	var env Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, CodeInvalidFilter, env.Error.Code)
	assert.Equal(t, "date_from", env.Error.Details["field"])
	assert.Equal(t, "not-a-date", env.Error.Details["value"])
}

func TestEnvelope_OmitsEmptyFields(t *testing.T) {
	env := Envelope{Error: Body{Code: CodeUnauthorized, Message: "denied"}}
	b, err := json.Marshal(env)
	require.NoError(t, err)
	got := string(b)
	assert.NotContains(t, got, "request_id", "request_id must be omitted when empty")
	assert.NotContains(t, got, "details", "details must be omitted when nil")
}

func TestCodes_Stable(t *testing.T) {
	cases := map[string]string{
		CodeInvalidRequest:         "invalid_request",
		CodeMissingQuery:           "missing_query",
		CodeInvalidURL:             "invalid_url",
		CodeInvalidFilter:          "invalid_filter",
		CodeUnauthorized:           "unauthorized",
		CodeForbidden:              "forbidden",
		CodeBlockedDomain:          "blocked_domain",
		CodeNotFound:               "not_found",
		CodeMethodNotAllowed:       "method_not_allowed",
		CodeUnsupportedContentType: "unsupported_content_type",
		CodeExtractionFailed:       "extraction_failed",
		CodeResponseTooLarge:       "response_too_large",
		CodeRateLimited:            "rate_limited",
		CodeUpstreamUnavailable:    "upstream_unavailable",
		CodeFetchTimeout:           "fetch_timeout",
		CodeInternalError:          "internal_error",
	}
	for got, want := range cases {
		assert.Equal(t, want, got)
	}
}
