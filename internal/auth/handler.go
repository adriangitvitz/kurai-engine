package auth

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/adriangitvitz/kurai/internal/apierror"
)

type AuthRequest struct {
	Key string `json:"key"`
}

type AuthResponse struct {
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	ExpiresAt string `json:"expires_at"`
}

type AuthHandler struct {
	store      *UserStore
	sessionMgr *SessionManager
	limiter    *RateLimiter
}

func NewAuthHandler(store *UserStore, sessionMgr *SessionManager, rateLimit int) *AuthHandler {
	return &AuthHandler{
		store:      store,
		sessionMgr: sessionMgr,
		limiter:    NewRateLimiter(rateLimit),
	}
}

func (h *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierror.Write(w, http.StatusMethodNotAllowed, apierror.CodeMethodNotAllowed, "POST method required", "")
		return
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip == "" {
		ip = r.RemoteAddr
	}
	if !h.limiter.Allow(ip) {
		apierror.Write(w, http.StatusTooManyRequests, apierror.CodeRateLimited, "too many auth attempts; try again later", "")
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Key == "" {
		apierror.Write(w, http.StatusBadRequest, apierror.CodeInvalidRequest, "valid JSON body with 'key' field required", "")
		return
	}

	user, ok := h.store.Authenticate(req.Key)
	if !ok {
		slog.Info("auth failed", "ip", ip)
		apierror.Write(w, http.StatusUnauthorized, apierror.CodeUnauthorized, "invalid API key", "")
		return
	}

	token, expiry := h.sessionMgr.CreateSession(user.ID)

	http.SetCookie(w, &http.Cookie{
		Name:     h.sessionMgr.CookieName(),
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.sessionMgr.Secure(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.sessionMgr.TTL().Seconds()),
	})

	slog.Info("auth success", "user", user.ID, "ip", ip)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(AuthResponse{
		UserID:    user.ID,
		Name:      user.Name,
		ExpiresAt: expiry.Format(time.RFC3339),
	})
}
