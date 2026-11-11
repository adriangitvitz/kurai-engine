# Error codes

Every Kurai HTTP error response uses the same JSON envelope:

```json
{
  "error": {
    "code": "missing_query",
    "message": "query parameter 'q' is required",
    "details": { }
  },
  "request_id": "req_..."
}
```

- `error.code` is a stable string. Machine clients (agents, MCP tools, scripts)
  should branch on `code`, not on the human-readable `message` or HTTP status.
- `error.message` is human-readable and may change across releases.
- `error.details` is optional, machine-readable extra context (e.g. which filter
  field was bad).
- `request_id` is included once request-ID middleware lands (Sprint 3.3); until
  then it's omitted.

Adding a new code is a backward-compatible change. Renaming or removing one is a
breaking change and must be called out in the release notes.

## Codes

| Code | HTTP status | Meaning |
|---|---|---|
| `invalid_request` | 400 | Generic malformed request (bad JSON, unknown enum, etc.) |
| `missing_query` | 400 | Required `q` query parameter is absent or empty |
| `invalid_url` | 400 | A URL parameter (e.g. `/extract?url=...`) failed to parse |
| `invalid_filter` | 400 | A filter value is malformed (e.g. `date_from` not a date) |
| `unauthorized` | 401 | No credentials, or credentials invalid/expired |
| `forbidden` | 403 | Authenticated, but not allowed to access the resource |
| `not_found` | 404 | Route doesn't exist |
| `method_not_allowed` | 405 | HTTP method not supported on this route |
| `response_too_large` | 413 | Upstream response exceeded the configured byte cap |
| `unsupported_content_type` | 415 | `/extract` got a non-HTML body (PDF, binary, ...) |
| `extraction_failed` | 422 | `/extract` fetched the page but couldn't isolate main content |
| `rate_limited` | 429 | Per-IP/user rate limit hit |
| `blocked_domain` | 451 | Target host is in the operator's blocklist |
| `internal_error` | 500 | Unhandled server-side error |
| `upstream_unavailable` | 502 | SearXNG or another upstream returned an error / no usable response |
| `fetch_timeout` | 504 | Upstream request exceeded the configured timeout |

## Auth-specific behavior

`401 Unauthorized` responses include a `WWW-Authenticate: Bearer realm="kurai", Cookie`
header so that compliant HTTP clients know both auth methods are accepted.
