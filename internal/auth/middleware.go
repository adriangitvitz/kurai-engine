package auth

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/adriangitvitz/kurai/internal/apierror"
)

type contextKey string

const userIDKey contextKey = "kurai_user_id"

func UserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok && id != "" {
		return id
	}
	return "anonymous"
}

func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func RequireAuth(sessionMgr *SessionManager, userStore *UserStore, bearerLimiter *RateLimiter, authEnabled bool) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !authEnabled {
				r = r.WithContext(withUserID(r.Context(), "anonymous"))
				next(w, r)
				return
			}

			if rawKey, ok := bearerKeyFromHeader(r.Header.Get("Authorization")); ok {
				ip := clientIP(r)
				if bearerLimiter.Limited(ip) {
					w.Header().Set("WWW-Authenticate", `Bearer realm="kurai"`)
					apierror.Write(w, http.StatusTooManyRequests, apierror.CodeRateLimited, "too many failed auth attempts; try again later", "")
					return
				}
				if userStore == nil {
					unauthorized(w, "bearer auth not configured")
					return
				}
				user, ok := userStore.Authenticate(rawKey)
				if !ok {
					bearerLimiter.RecordFailure(ip)
					unauthorized(w, "invalid API key")
					return
				}
				r = r.WithContext(withUserID(r.Context(), user.ID))
				next(w, r)
				return
			}

			if r.Header.Get("Authorization") != "" {
				unauthorized(w, "unsupported Authorization scheme; use Bearer")
				return
			}

			cookie, err := r.Cookie(sessionMgr.CookieName())
			if err != nil {
				unauthorized(w, "authentication required")
				return
			}

			userID, err := sessionMgr.ValidateSession(cookie.Value)
			if err != nil {
				unauthorized(w, "invalid or expired session")
				return
			}

			r = r.WithContext(withUserID(r.Context(), userID))
			next(w, r)
		}
	}
}

func bearerKeyFromHeader(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	key := strings.TrimSpace(header[len(prefix):])
	if key == "" {
		return "", false
	}
	return key, true
}

func clientIP(r *http.Request) string {
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return r.RemoteAddr
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="kurai", Cookie`)
	apierror.Write(w, http.StatusUnauthorized, apierror.CodeUnauthorized, msg, "")
}
