package server

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/adriangitvitz/kurai/internal/answers"
	"github.com/adriangitvitz/kurai/internal/auth"
	"github.com/adriangitvitz/kurai/internal/bookmarks"
	"github.com/adriangitvitz/kurai/internal/cache"
	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/extract"
	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/internal/searxng"
	"github.com/adriangitvitz/kurai/internal/tavily"
)

type requestIDKey struct{}

func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func NewRequestID() string { return generateRequestID() }

func generateRequestID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	return "req_" + base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
}

type Server struct {
	config      *config.Config
	httpServer  *http.Server
	mux         *http.ServeMux
	requireAuth func(http.HandlerFunc) http.HandlerFunc
	searxClient *searxng.Client
	tavily      *tavily.Client
	answers     *answers.Resolver
	bookmarks   bookmarks.Store
	ranker      *ranker.Ranker
	blocklist   BlocklistManager
	cache       *cache.SearchCache
	extractor   Extractor

	engineStats        *engineStats
	fusedExtractBudget time.Duration
	prober             *livenessProber
	probeDrop          bool
	startTime          time.Time
	pageSize           int
	prefetchPages      int
	sessionMgr         *auth.SessionManager
	userStore          *auth.UserStore
	authEnabled        bool
}

func New(cfg *config.Config, searxClient *searxng.Client, rankEngine *ranker.Ranker, bl BlocklistManager) *Server {
	s := &Server{
		config:        cfg,
		searxClient:   searxClient,
		ranker:        rankEngine,
		blocklist:     bl,
		cache:         cache.NewSearchCache(cfg.Server.CacheTTL),
		startTime:     time.Now(),
		pageSize:      cfg.Server.PageSize,
		prefetchPages: cfg.Server.PrefetchPages,
		authEnabled:   cfg.Auth.Enabled,
	}

	if cfg.Tavily.Enabled {
		s.tavily = tavily.NewClient(&cfg.Tavily, &cfg.Privacy)
	}

	if cfg.Answers.Enabled {
		s.answers = answers.NewResolver(&cfg.Answers, s.cache)
	}

	if cfg.Bookmarks.Enabled {
		store, err := bookmarks.OpenSQLite(cfg.Bookmarks.Path)
		if err != nil {
			slog.Error("bookmarks: failed to open store, board disabled",
				"path", cfg.Bookmarks.Path, "error", err)
		} else {
			s.bookmarks = store
			slog.Info("bookmarks enabled", "path", cfg.Bookmarks.Path)
		}
	}

	if cfg.Extract.Enabled {
		s.extractor = extract.NewFetcher(extract.Config{
			Timeout:              cfg.Extract.Timeout,
			MaxBytes:             cfg.Extract.MaxBytes,
			MaxRedirects:         cfg.Extract.MaxRedirects,
			UserAgent:            cfg.Extract.UserAgent,
			AllowPrivateNetworks: cfg.Extract.AllowPrivateNetworks,
		})
		s.fusedExtractBudget = cfg.Extract.FusedBudget
	}

	s.engineStats = newEngineStats()

	if cfg.Probe.Enabled {
		s.prober = newLivenessProber(cfg.Probe)
		s.probeDrop = cfg.Probe.DropUnreachable
	}

	if cfg.Auth.Enabled {
		secret, err := base64.StdEncoding.DecodeString(cfg.Auth.ServerSecret)
		if err != nil || len(secret) < 32 {
			slog.Error("auth: server_secret must be a base64-encoded string of at least 32 bytes")
			s.authEnabled = false
		} else {
			s.sessionMgr = auth.NewSessionManager(secret, cfg.Auth.SessionTTL, cfg.Auth.CookieSecure)

			s.userStore = auth.NewUserStore()
			if err := s.userStore.LoadFromFile(cfg.Auth.UsersFile); err != nil {
				slog.Error("auth: failed to load users file", "path", cfg.Auth.UsersFile, "error", err)
				s.authEnabled = false
			} else {
				slog.Info("auth enabled", "users", s.userStore.UserCount())
			}
		}
	}

	if s.sessionMgr == nil {
		s.sessionMgr = auth.NewSessionManager([]byte("disabled-not-used-placeholder!!"), 8*time.Hour, false)
	}

	bearerLimiter := auth.NewRateLimiter(cfg.Auth.BearerRateLimit)
	s.requireAuth = auth.RequireAuth(s.sessionMgr, s.userStore, bearerLimiter, s.authEnabled)
	requireAuth := s.requireAuth

	mux := http.NewServeMux()
	s.mux = mux

	mux.HandleFunc("/health", s.withMiddleware(s.handleHealth))

	if oa, err := newOpenAPIHandlers(); err == nil {
		mux.HandleFunc("/openapi.yaml", s.withMiddleware(oa.yaml()))
		mux.HandleFunc("/openapi.json", s.withMiddleware(oa.json()))
	} else {
		slog.Warn("openapi: failed to initialize handlers", "error", err)
	}

	if s.authEnabled {
		authHandler := auth.NewAuthHandler(s.userStore, s.sessionMgr, cfg.Auth.RateLimit)
		mux.Handle("/auth", s.withMiddlewareHandler(authHandler))
	}

	mux.HandleFunc("/search", s.withMiddleware(requireAuth(s.handleSearch)))
	mux.HandleFunc("/blocklist/reload", s.withMiddleware(requireAuth(s.handleBlocklistReload)))
	mux.HandleFunc("/stats/engines", s.withMiddleware(requireAuth(s.handleEngineStats)))

	if s.bookmarks != nil {
		mux.HandleFunc("/bookmarks", s.withMiddleware(requireAuth(s.handleBookmarks)))
	}

	if cfg.Images.Enabled {
		mux.HandleFunc("/images", s.withMiddleware(requireAuth(s.handleImages)))
	}

	if cfg.News.Enabled {
		mux.HandleFunc("/news", s.withMiddleware(requireAuth(s.handleNews)))
		mux.HandleFunc("/news/trending", s.withMiddleware(requireAuth(s.handleNewsTrending)))
		mux.HandleFunc("/news/feed", s.withMiddleware(requireAuth(s.handleNewsFeed)))
	}

	if cfg.Extract.Enabled {
		mux.HandleFunc("/extract", s.withMiddleware(requireAuth(s.handleExtract)))
	}

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return s
}

func (s *Server) ListenAndServe() error {
	slog.Info("starting server", "port", s.config.Server.Port)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.httpServer.Handler.ServeHTTP(w, r)
}

func (s *Server) withMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
		}
		w.Header().Set("X-Request-ID", reqID)
		r = r.WithContext(context.WithValue(r.Context(), requestIDKey{}, reqID))

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")

		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "interest-cohort=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'none'; style-src 'unsafe-inline'")

		if s.authEnabled {
			origin := r.Header.Get("Origin")
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Mcp-Session-Id, MCP-Protocol-Version")
		w.Header().Set("Access-Control-Expose-Headers", "Mcp-Session-Id, MCP-Protocol-Version, X-Request-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		handler(w, r)

		duration := time.Since(start)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", duration,
			"request_id", reqID,
		)
	}
}

func (s *Server) withMiddlewareHandler(handler http.Handler) http.HandlerFunc {
	return s.withMiddleware(handler.ServeHTTP)
}

func (s *Server) MountAuthed(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, s.withMiddlewareHandler(http.HandlerFunc(s.requireAuth(handler.ServeHTTP))))
}
