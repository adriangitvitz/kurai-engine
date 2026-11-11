package apierror

import (
	"encoding/json"
	"net/http"
)

const (
	CodeInvalidRequest         = "invalid_request"
	CodeMissingQuery           = "missing_query"
	CodeInvalidURL             = "invalid_url"
	CodeInvalidFilter          = "invalid_filter"
	CodeUnauthorized           = "unauthorized"
	CodeForbidden              = "forbidden"
	CodeBlockedDomain          = "blocked_domain"
	CodeNotFound               = "not_found"
	CodeMethodNotAllowed       = "method_not_allowed"
	CodeUnsupportedContentType = "unsupported_content_type"
	CodeExtractionFailed       = "extraction_failed"
	CodeResponseTooLarge       = "response_too_large"
	CodeRateLimited            = "rate_limited"
	CodeUpstreamUnavailable    = "upstream_unavailable"
	CodeFetchTimeout           = "fetch_timeout"
	CodeInternalError          = "internal_error"
)

type Envelope struct {
	Error     Body   `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

type Body struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func Write(w http.ResponseWriter, status int, code, message, requestID string) {
	WriteWithDetails(w, status, code, message, nil, requestID)
}

func WriteWithDetails(w http.ResponseWriter, status int, code, message string, details map[string]any, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{
		Error:     Body{Code: code, Message: message, Details: details},
		RequestID: requestID,
	})
}
