package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/adriangitvitz/kurai/internal/auth"
	"github.com/adriangitvitz/kurai/internal/blocklist"
	"github.com/adriangitvitz/kurai/internal/config"
	kuraimcp "github.com/adriangitvitz/kurai/internal/mcp"
	"github.com/adriangitvitz/kurai/internal/ranker"
	"github.com/adriangitvitz/kurai/internal/reranker"
	"github.com/adriangitvitz/kurai/internal/searxng"
	"github.com/adriangitvitz/kurai/internal/server"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func logLevel() slog.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("KURAI_LOG_LEVEL"))) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "gen-secret":
			secret := make([]byte, 32)
			if _, err := rand.Read(secret); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(base64.StdEncoding.EncodeToString(secret))
			return
		case "gen-key":
			runGenKey(os.Args[2:])
			return
		case "mcp":
			runMCPSubcommand(os.Args[2:])
			return
		}
	}

	configPath := flag.String("config", "config.yaml", "Path to config file")
	forceRefresh := flag.Bool("force-refresh-blocklists", false, "Force refresh blocklists from remote sources")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel(),
	}))
	slog.SetDefault(logger)

	slog.Info("starting kurai", "version", version, "build_time", buildTime)

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	blManager := blocklist.NewManager(&cfg.Blocklists, &cfg.Privacy)

	if *forceRefresh {
		slog.Info("forcing blocklist refresh from remote sources")
		cfg.Blocklists.Mode = "online"
	}

	loadCtx, loadCancel := context.WithTimeout(ctx, 2*time.Minute)
	if err := blManager.Load(loadCtx); err != nil {
		slog.Warn("failed to load blocklists", "error", err)
	}
	loadCancel()

	slog.Info("blocklists loaded",
		"ads_domains", blManager.AdsCount(),
		"tranco_domains", blManager.TrancoCount())

	if err := ranker.LoadContentFarms(cfg.Blocklists.ContentFarmPath); err != nil {
		slog.Warn("failed to load content farms", "error", err)
	}

	searxClient := searxng.NewClient(&cfg.SearXNG, &cfg.Privacy)

	slog.Info("privacy settings",
		"doh_enabled", cfg.Privacy.DoH.Enabled,
		"doh_resolver", cfg.Privacy.DoH.ResolverURL,
		"searxng_url", cfg.SearXNG.URL)

	rankEngine := ranker.New(&cfg.Ranking, blManager)

	if rr := buildReranker(&cfg.Reranker); rr != nil {
		rankEngine.SetReranker(rr, cfg.Reranker.TopN, cfg.Reranker.BlendAlpha)
		slog.Info("reranker enabled", "provider", cfg.Reranker.Provider, "url", cfg.Reranker.URL)
	}

	srv := server.New(cfg, searxClient, rankEngine, blManager)

	if cfg.MCP.Enabled {
		mcpOpts := &kuraimcp.Options{KeepAlive: cfg.MCP.KeepAlive}
		mcpHTTP := kuraimcp.NewHTTPHandler(func(*http.Request) *mcpsdk.Server {
			return kuraimcp.NewToolServer(srv, version, mcpOpts)
		}, cfg.MCP.SessionTimeout)
		srv.MountAuthed("/mcp", mcpHTTP)
		slog.Info("mcp http transport mounted",
			"keepalive", cfg.MCP.KeepAlive,
			"session_timeout", cfg.MCP.SessionTimeout)
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			slog.Error("server error", "error", err)
			cancel()
		}
	}()

	slog.Info("server started", "port", cfg.Server.Port, "auth_enabled", cfg.Auth.Enabled)

	if cfg.Blocklists.RefreshHours > 0 {
		go scheduleBlocklistRefresh(ctx, blManager, cfg.Blocklists.RefreshHours)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}

	slog.Info("shutdown complete")
}

func runGenKey(args []string) {
	fs := flag.NewFlagSet("gen-key", flag.ExitOnError)
	userID := fs.String("user", "", "User ID (required, e.g. \"alice\")")
	name := fs.String("name", "", "Display name (defaults to user ID)")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}
	if *userID == "" {
		fmt.Fprintln(os.Stderr, "kurai gen-key: --user is required")
		fmt.Fprintln(os.Stderr, "  example: kurai gen-key --user alice --name \"Alice Smith\"")
		os.Exit(2)
	}
	displayName := *name
	if displayName == "" {
		displayName = *userID
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating key: %v\n", err)
		os.Exit(1)
	}
	rawKey := base64.RawURLEncoding.EncodeToString(raw)
	hashHex := auth.HashAPIKeyHex(rawKey)

	fmt.Fprintln(os.Stderr, "# Raw API key (share this ONCE with the user - it cannot be recovered):")
	fmt.Println(rawKey)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "# Append the following to your users.yaml:")
	fmt.Fprintln(os.Stderr, "users:")
	fmt.Fprintf(os.Stderr, "- id: %s\n", *userID)
	fmt.Fprintf(os.Stderr, "  name: %q\n", displayName)
	fmt.Fprintf(os.Stderr, "  key_hash: %s\n", hashHex)
	fmt.Fprintln(os.Stderr, "  enabled: true")
}

func runMCPSubcommand(args []string) {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "Path to config file")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing mcp flags: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel()}))
	slog.SetDefault(logger)

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	bl := blocklist.NewManager(&cfg.Blocklists, &cfg.Privacy)
	bl.StartBackgroundLoad(ctx)

	if err := ranker.LoadContentFarms(cfg.Blocklists.ContentFarmPath); err != nil {
		slog.Warn("failed to load content farms", "error", err)
	}

	searxClient := searxng.NewClient(&cfg.SearXNG, &cfg.Privacy)
	rankEngine := ranker.New(&cfg.Ranking, bl)
	srv := server.New(cfg, searxClient, rankEngine, bl)

	mcpsrv := kuraimcp.NewToolServer(srv, version, &kuraimcp.Options{KeepAlive: cfg.MCP.KeepAlive})
	slog.Info("kurai MCP stdio server starting", "version", version)
	if err := mcpsrv.Run(ctx, &mcpsdk.StdioTransport{}); err != nil {
		slog.Error("MCP server exited with error", "error", err)
		os.Exit(1)
	}
}

func buildReranker(cfg *config.RerankerConfig) reranker.Reranker {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	switch cfg.Provider {
	case "tei":
		if cfg.URL == "" {
			slog.Warn("reranker.provider=tei but reranker.url is empty; reranker disabled")
			return nil
		}
		return reranker.NewTEIClient(cfg.URL, timeout)
	case "noop":
		return reranker.Noop{}
	default:
		slog.Warn("unknown reranker.provider; reranker disabled", "provider", cfg.Provider)
		return nil
	}
}

func scheduleBlocklistRefresh(ctx context.Context, manager *blocklist.Manager, hours int) {
	ticker := time.NewTicker(time.Duration(hours) * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			slog.Info("refreshing blocklists...")
			refreshCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			if err := manager.Reload(refreshCtx); err != nil {
				slog.Error("blocklist refresh failed", "error", err)
			} else {
				slog.Info("blocklists refreshed",
					"ads_domains", manager.AdsCount(),
					"tranco_domains", manager.TrancoCount())
			}
			cancel()
		}
	}
}
