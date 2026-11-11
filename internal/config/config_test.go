package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ValidYAMLConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
searxng:
  url: "http://localhost:9999"
  timeout: 10s

server:
  port: 9080

ranking:
  weights:
    title_match: 25
    url_match: 15
  penalties:
    blocklist: -60
    content_farm: -40

blocklists:
  easylist: true
  stevenblack: false
  tranco: true
  custom: "./custom.txt"

probe:
  enabled: true
  top_n: 10
  timeout: 3s
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "http://localhost:9999", cfg.SearXNG.URL)
	assert.Equal(t, 10*time.Second, cfg.SearXNG.Timeout)
	assert.Equal(t, 9080, cfg.Server.Port)
	assert.Equal(t, 25, cfg.Ranking.Weights.TitleMatch)
	assert.Equal(t, 15, cfg.Ranking.Weights.URLMatch)
	assert.Equal(t, -60, cfg.Ranking.Penalties.Blocklist)
	assert.Equal(t, -40, cfg.Ranking.Penalties.ContentFarm)
	assert.True(t, cfg.Blocklists.EasyList)
	assert.False(t, cfg.Blocklists.StevenBlack)
	assert.True(t, cfg.Blocklists.Tranco)
	assert.Equal(t, "./custom.txt", cfg.Blocklists.CustomPath)
	assert.True(t, cfg.Probe.Enabled)
	assert.Equal(t, 10, cfg.Probe.TopN)
	assert.Equal(t, 3*time.Second, cfg.Probe.Timeout)
}

func TestLoad_DefaultsWhenConfigMissing(t *testing.T) {
	cfg, err := Load("")
	require.NoError(t, err)

	assert.Equal(t, "http://localhost:8888", cfg.SearXNG.URL)
	assert.Equal(t, 5*time.Second, cfg.SearXNG.Timeout)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 20, cfg.Ranking.Weights.TitleMatch)
	assert.Equal(t, 10, cfg.Ranking.Weights.URLMatch)
	assert.Equal(t, 5, cfg.Ranking.Weights.EngineAgreement)
	assert.Equal(t, 15, cfg.Ranking.Weights.TrancoTop10k)
	assert.Equal(t, 10, cfg.Ranking.Weights.TrancoTop100k)
	assert.Equal(t, 5, cfg.Ranking.Weights.TrancoTop1M)
	assert.Equal(t, 5, cfg.Ranking.Weights.Freshness)
	assert.Equal(t, -50, cfg.Ranking.Penalties.Blocklist)
	assert.Equal(t, -30, cfg.Ranking.Penalties.ContentFarm)
	assert.Equal(t, -10, cfg.Ranking.Penalties.ExcessiveParams)
	assert.Equal(t, 5, cfg.Ranking.Penalties.ParamThreshold)
	assert.True(t, cfg.Blocklists.EasyList)
	assert.True(t, cfg.Blocklists.StevenBlack)
	assert.True(t, cfg.Blocklists.Tranco)
	assert.Equal(t, "./blocklists/custom.txt", cfg.Blocklists.CustomPath)
	assert.Equal(t, 24, cfg.Blocklists.RefreshHours)
	assert.True(t, cfg.Probe.Enabled)
	assert.Equal(t, 20, cfg.Probe.TopN)
	assert.Equal(t, 2*time.Second, cfg.Probe.Timeout)
	assert.True(t, cfg.Probe.DropUnreachable)
	assert.False(t, cfg.Probe.AllowPrivateNetworks)
}

func TestLoad_EnvVarOverrides(t *testing.T) {
	t.Setenv("KURAI_SEARXNG_URL", "http://override:1234")
	t.Setenv("KURAI_PORT", "7777")
	t.Setenv("KURAI_PROBE_ENABLED", "true")

	cfg, err := Load("")
	require.NoError(t, err)

	assert.Equal(t, "http://override:1234", cfg.SearXNG.URL)
	assert.Equal(t, 7777, cfg.Server.Port)
	assert.True(t, cfg.Probe.Enabled)
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	require.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config")
}

func TestLoad_PartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
searxng:
  url: "http://custom:5555"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "http://custom:5555", cfg.SearXNG.URL)
	assert.Equal(t, 5*time.Second, cfg.SearXNG.Timeout)
	assert.Equal(t, 8080, cfg.Server.Port)
}

func TestLoad_FileNotFound(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	require.NoError(t, err)

	assert.Equal(t, "http://localhost:8888", cfg.SearXNG.URL)
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("KURAI_PORT", "invalid")

	cfg, err := Load("")
	require.NoError(t, err)

	assert.Equal(t, 8080, cfg.Server.Port)
}
