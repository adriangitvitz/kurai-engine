package blocklist

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/adriangitvitz/kurai/internal/config"
	"github.com/adriangitvitz/kurai/internal/privacy"
)

const (
	easyListURL    = "https://easylist.to/easylist/easylist.txt"
	stevenBlackURL = "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts"
	trancoURL      = "https://tranco-list.eu/top-1m.csv.zip"
)

type LoadState int32

const (
	StateEmpty   LoadState = 0
	StateLoading LoadState = 1
	StateReady   LoadState = 2
	StateError   LoadState = 3
)

func (s LoadState) String() string {
	switch s {
	case StateEmpty:
		return "empty"
	case StateLoading:
		return "loading"
	case StateReady:
		return "ready"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

type Manager struct {
	config     *config.BlocklistConfig
	adsDomains *DomainStore
	customList *DomainStore
	tranco     *TrancoStore
	lastUpdate time.Time
	mu         sync.RWMutex
	httpClient *http.Client

	state  atomic.Int32
	loaded sync.Once
}

type Status struct {
	AdsCount    int       `json:"ads_count"`
	CustomCount int       `json:"custom_count"`
	TrancoCount int       `json:"tranco_count"`
	LastUpdate  time.Time `json:"last_update"`
	LoadState   LoadState `json:"load_state"`
}

func NewManager(cfg *config.BlocklistConfig, privacyCfg *config.PrivacyConfig) *Manager {
	var httpClient *http.Client

	if privacyCfg != nil && privacyCfg.DoH.Enabled {
		dohClient, err := privacy.NewDoHClient(privacyCfg.DoH.ResolverURL)
		if err == nil {
			httpClient = dohClient
		}
	}

	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &Manager{
		config:     cfg,
		adsDomains: NewDomainStore(500000),
		customList: NewDomainStore(1000),
		tranco:     NewTrancoStore(),
		httpClient: httpClient,
	}
}

func (m *Manager) State() LoadState {
	return LoadState(m.state.Load())
}

func (m *Manager) SetState(s LoadState) {
	m.state.Store(int32(s))
}

func (m *Manager) IsBlocked(domain string) bool {
	if LoadState(m.state.Load()) < StateReady {
		return false
	}
	return m.adsDomains.Contains(domain) || m.customList.Contains(domain)
}

func (m *Manager) GetTrancoRank(domain string) int {
	if LoadState(m.state.Load()) < StateReady {
		return 0
	}
	return m.tranco.GetRank(domain)
}

func (m *Manager) AdsCount() int {
	return m.adsDomains.Count()
}

func (m *Manager) TrancoCount() int {
	return m.tranco.Count()
}

func (m *Manager) LastUpdate() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastUpdate
}

func (m *Manager) SetLastUpdate(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastUpdate = t
}

func (m *Manager) Status() Status {
	return Status{
		AdsCount:    m.adsDomains.Count(),
		CustomCount: m.customList.Count(),
		TrancoCount: m.tranco.Count(),
		LastUpdate:  m.LastUpdate(),
		LoadState:   m.State(),
	}
}

func (m *Manager) ensureCacheDir() error {
	if m.config.CachePath == "" {
		return nil
	}
	return os.MkdirAll(m.config.CachePath, 0755)
}

func (m *Manager) getCachePath(name string) string {
	if m.config.CachePath == "" {
		return ""
	}
	return filepath.Join(m.config.CachePath, name)
}

func (m *Manager) isCacheValid(cachePath string) bool {
	if cachePath == "" {
		return false
	}
	info, err := os.Stat(cachePath)
	if err != nil {
		return false
	}
	age := time.Since(info.ModTime())
	maxAge := time.Duration(m.config.RefreshHours) * time.Hour
	return age < maxAge
}

func (m *Manager) loadFromCache(cachePath string) (io.ReadCloser, error) {
	return os.Open(cachePath)
}

func (m *Manager) saveToCache(cachePath string, data []byte) error {
	if cachePath == "" {
		return nil
	}
	return os.WriteFile(cachePath, data, 0644)
}

func (m *Manager) Load(ctx context.Context) error {
	if err := m.ensureCacheDir(); err != nil {
		slog.Warn("failed to create blocklist cache directory", "error", err)
	}

	if err := m.LoadCustom(); err != nil {
		return fmt.Errorf("loading custom blocklist: %w", err)
	}

	mode := m.config.Mode
	if mode == "" {
		mode = "cache-first"
	}

	if m.config.EasyList {
		if err := m.loadWithCache(ctx, "easylist.txt", easyListURL, m.loadEasyListReader, mode); err != nil {
			if mode == "offline" {
				slog.Warn("failed to load EasyList from cache", "error", err)
			} else {
				return fmt.Errorf("loading EasyList: %w", err)
			}
		}
	}

	if m.config.StevenBlack {
		if err := m.loadWithCache(ctx, "stevenblack.txt", stevenBlackURL, m.loadStevenBlackReader, mode); err != nil {
			if mode == "offline" {
				slog.Warn("failed to load StevenBlack from cache", "error", err)
			} else {
				return fmt.Errorf("loading StevenBlack: %w", err)
			}
		}
	}

	if m.config.Tranco {
		if err := m.loadWithCache(ctx, "tranco.csv", trancoURL, m.loadTrancoReader, mode); err != nil {
			if mode == "offline" {
				slog.Warn("failed to load Tranco from cache", "error", err)
			} else {
				return fmt.Errorf("loading Tranco: %w", err)
			}
		}
	}

	m.SetLastUpdate(time.Now())
	m.state.Store(int32(StateReady))
	return nil
}

func (m *Manager) StartBackgroundLoad(ctx context.Context) {
	m.loaded.Do(func() {
		m.state.Store(int32(StateLoading))
		go func() {
			if err := m.Load(ctx); err != nil {
				slog.Warn("background blocklist load failed", "error", err)
				m.state.Store(int32(StateError))
				return
			}
			slog.Info("background blocklist load complete",
				"ads_domains", m.AdsCount(),
				"tranco_domains", m.TrancoCount())
		}()
	})
}

func (m *Manager) loadWithCache(ctx context.Context, cacheName, url string, loader func(io.Reader) error, mode string) error {
	cachePath := m.getCachePath(cacheName)

	switch mode {
	case "offline":
		if cachePath == "" {
			return fmt.Errorf("cache path not configured for offline mode")
		}
		reader, err := m.loadFromCache(cachePath)
		if err != nil {
			return fmt.Errorf("cache not available: %w", err)
		}
		defer reader.Close()
		slog.Info("loading blocklist from cache", "name", cacheName)
		return loader(reader)

	case "cache-first":
		if m.isCacheValid(cachePath) {
			reader, err := m.loadFromCache(cachePath)
			if err == nil {
				defer reader.Close()
				slog.Info("loading blocklist from cache", "name", cacheName)
				return loader(reader)
			}
		}
		fallthrough

	case "online":
		body, err := m.fetchURL(ctx, url)
		if err != nil {
			if mode == "cache-first" && cachePath != "" {
				reader, cacheErr := m.loadFromCache(cachePath)
				if cacheErr == nil {
					defer reader.Close()
					slog.Warn("using stale cache after download failed", "name", cacheName, "error", err)
					return loader(reader)
				}
			}
			return err
		}
		defer body.Close()

		data, err := io.ReadAll(body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		if err := m.saveToCache(cachePath, data); err != nil {
			slog.Warn("failed to cache blocklist", "name", cacheName, "error", err)
		} else {
			slog.Info("cached blocklist", "name", cacheName)
		}

		return loader(strings.NewReader(string(data)))

	default:
		return fmt.Errorf("unknown blocklist mode: %s", mode)
	}
}

func (m *Manager) loadEasyListReader(r io.Reader) error {
	return ParseEasyList(r, m.adsDomains)
}

func (m *Manager) loadStevenBlackReader(r io.Reader) error {
	return ParseHosts(r, m.adsDomains)
}

func (m *Manager) loadTrancoReader(r io.Reader) error {
	return m.tranco.ParseAuto(r)
}

func (m *Manager) Reload(ctx context.Context) error {
	return m.Load(ctx)
}

func (m *Manager) LoadCustom() error {
	if m.config.CustomPath == "" {
		return nil
	}

	file, err := os.Open(m.config.CustomPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		m.customList.Add(line)
	}

	return scanner.Err()
}

func (m *Manager) loadEasyListFrom(ctx context.Context, url string) error {
	body, err := m.fetchURL(ctx, url)
	if err != nil {
		return err
	}
	defer body.Close()

	return ParseEasyList(body, m.adsDomains)
}

func (m *Manager) loadStevenBlackFrom(ctx context.Context, url string) error {
	body, err := m.fetchURL(ctx, url)
	if err != nil {
		return err
	}
	defer body.Close()

	return ParseHosts(body, m.adsDomains)
}

func (m *Manager) loadTrancoFrom(ctx context.Context, url string) error {
	body, err := m.fetchURL(ctx, url)
	if err != nil {
		return err
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	return m.tranco.ParseZip(strings.NewReader(string(data)), int64(len(data)))
}

func (m *Manager) loadTrancoCSVFrom(ctx context.Context, url string) error {
	body, err := m.fetchURL(ctx, url)
	if err != nil {
		return err
	}
	defer body.Close()

	return m.tranco.ParseCSV(body)
}

func (m *Manager) fetchURL(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching URL: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return resp.Body, nil
}
