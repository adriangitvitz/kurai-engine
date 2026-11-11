package mcp

import (
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewHTTPHandler(getServer func(*http.Request) *mcpsdk.Server, sessionTimeout time.Duration) http.Handler {
	h := mcpsdk.NewStreamableHTTPHandler(getServer, &mcpsdk.StreamableHTTPOptions{
		SessionTimeout: sessionTimeout,
	})

	var warnOnce sync.Once
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := http.NewResponseController(w).SetWriteDeadline(time.Time{}); err != nil &&
			!errors.Is(err, http.ErrNotSupported) {
			warnOnce.Do(func() {
				slog.Warn("mcp: could not clear write deadline; long-lived SSE streams "+
					"will be cut by server.write_timeout", "error", err)
			})
		}
		h.ServeHTTP(w, r)
	})
}
