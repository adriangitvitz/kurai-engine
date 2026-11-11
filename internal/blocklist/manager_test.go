package blocklist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/config"
)

func TestManager_NewManager(t *testing.T) {
	cfg := &config.BlocklistConfig{
		EasyList:    true,
		StevenBlack: true,
		Tranco:      true,
		CustomPath:  "./custom.txt",
	}

	m := NewManager(cfg, nil)
	require.NotNil(t, m)
	assert.Equal(t, 0, m.AdsCount())
	assert.Equal(t, 0, m.TrancoCount())
}

func TestManager_IsBlocked(t *testing.T) {
	cfg := &config.BlocklistConfig{}
	m := NewManager(cfg, nil)

	m.adsDomains.Add("ads.example.com")
	m.adsDomains.Add("tracker.net")
	m.SetState(StateReady)

	assert.True(t, m.IsBlocked("ads.example.com"))
	assert.True(t, m.IsBlocked("tracker.net"))
	assert.False(t, m.IsBlocked("good.com"))
}

func TestManager_IsBlocked_CustomList(t *testing.T) {
	cfg := &config.BlocklistConfig{}
	m := NewManager(cfg, nil)

	m.customList.Add("my-blocked.com")
	m.SetState(StateReady)

	assert.True(t, m.IsBlocked("my-blocked.com"))
	assert.False(t, m.IsBlocked("not-blocked.com"))
}

func TestManager_GetTrancoRank(t *testing.T) {
	cfg := &config.BlocklistConfig{}
	m := NewManager(cfg, nil)

	m.tranco.Set("google.com", 1)
	m.tranco.Set("github.com", 100)
	m.SetState(StateReady)

	assert.Equal(t, 1, m.GetTrancoRank("google.com"))
	assert.Equal(t, 100, m.GetTrancoRank("github.com"))
	assert.Equal(t, 0, m.GetTrancoRank("unknown.com"))
}

func TestManager_LoadCustomBlocklist(t *testing.T) {
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "custom.txt")

	content := `# Custom blocklist
custom-ads.com
custom-tracker.net
www.custom-spam.org
`
	err := os.WriteFile(customPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg := &config.BlocklistConfig{
		CustomPath: customPath,
	}
	m := NewManager(cfg, nil)

	err = m.LoadCustom()
	require.NoError(t, err)
	m.SetState(StateReady)

	assert.True(t, m.IsBlocked("custom-ads.com"))
	assert.True(t, m.IsBlocked("custom-tracker.net"))
	assert.True(t, m.IsBlocked("custom-spam.org"))
	assert.Equal(t, 3, m.customList.Count())
}

func TestManager_LoadCustomBlocklist_NotFound(t *testing.T) {
	cfg := &config.BlocklistConfig{
		CustomPath: "/nonexistent/path/custom.txt",
	}
	m := NewManager(cfg, nil)

	err := m.LoadCustom()
	require.NoError(t, err)
	assert.Equal(t, 0, m.customList.Count())
}

func TestManager_LoadEasyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`||ads.test.com^
||tracker.test.net^
||spam.test.io^
`))
	}))
	defer server.Close()

	cfg := &config.BlocklistConfig{
		EasyList: true,
	}
	m := NewManager(cfg, nil)

	ctx := context.Background()
	err := m.loadEasyListFrom(ctx, server.URL)
	require.NoError(t, err)
	m.SetState(StateReady)

	assert.True(t, m.IsBlocked("ads.test.com"))
	assert.True(t, m.IsBlocked("tracker.test.net"))
	assert.True(t, m.IsBlocked("spam.test.io"))
}

func TestManager_LoadStevenBlack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`# StevenBlack hosts
0.0.0.0 ads.stevenblack.com
0.0.0.0 tracker.stevenblack.net
`))
	}))
	defer server.Close()

	cfg := &config.BlocklistConfig{
		StevenBlack: true,
	}
	m := NewManager(cfg, nil)

	ctx := context.Background()
	err := m.loadStevenBlackFrom(ctx, server.URL)
	require.NoError(t, err)
	m.SetState(StateReady)

	assert.True(t, m.IsBlocked("ads.stevenblack.com"))
	assert.True(t, m.IsBlocked("tracker.stevenblack.net"))
}

func TestManager_LoadTranco(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`1,top1.com
2,top2.com
100,top100.com
`))
	}))
	defer server.Close()

	cfg := &config.BlocklistConfig{
		Tranco: true,
	}
	m := NewManager(cfg, nil)

	ctx := context.Background()
	err := m.loadTrancoCSVFrom(ctx, server.URL)
	require.NoError(t, err)
	m.SetState(StateReady)

	assert.Equal(t, 1, m.GetTrancoRank("top1.com"))
	assert.Equal(t, 2, m.GetTrancoRank("top2.com"))
	assert.Equal(t, 100, m.GetTrancoRank("top100.com"))
}

func TestManager_Load_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte("||slow.com^"))
	}))
	defer server.Close()

	cfg := &config.BlocklistConfig{
		EasyList: true,
	}
	m := NewManager(cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := m.loadEasyListFrom(ctx, server.URL)
	assert.Error(t, err)
}

func TestManager_Reload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("||newdomain.com^"))
	}))
	defer server.Close()

	cfg := &config.BlocklistConfig{
		EasyList: true,
	}
	m := NewManager(cfg, nil)

	ctx := context.Background()
	err := m.loadEasyListFrom(ctx, server.URL)
	require.NoError(t, err)
	m.SetState(StateReady)
	assert.True(t, m.IsBlocked("newdomain.com"))

	err = m.loadEasyListFrom(ctx, server.URL)
	require.NoError(t, err)
	assert.True(t, m.IsBlocked("newdomain.com"))
}

func TestManager_AdsCount(t *testing.T) {
	cfg := &config.BlocklistConfig{}
	m := NewManager(cfg, nil)

	m.adsDomains.Add("ads1.com")
	m.adsDomains.Add("ads2.com")
	m.adsDomains.Add("ads3.com")

	assert.Equal(t, 3, m.AdsCount())
}

func TestManager_TrancoCount(t *testing.T) {
	cfg := &config.BlocklistConfig{}
	m := NewManager(cfg, nil)

	m.tranco.Set("site1.com", 1)
	m.tranco.Set("site2.com", 2)

	assert.Equal(t, 2, m.TrancoCount())
}

func TestManager_LastUpdate(t *testing.T) {
	cfg := &config.BlocklistConfig{}
	m := NewManager(cfg, nil)

	assert.True(t, m.LastUpdate().IsZero())

	now := time.Now()
	m.SetLastUpdate(now)
	assert.Equal(t, now, m.LastUpdate())
}

func TestManager_Status(t *testing.T) {
	cfg := &config.BlocklistConfig{
		EasyList:    true,
		StevenBlack: false,
		Tranco:      true,
	}
	m := NewManager(cfg, nil)

	m.adsDomains.Add("test.com")
	m.tranco.Set("google.com", 1)

	status := m.Status()
	assert.Equal(t, 1, status.AdsCount)
	assert.Equal(t, 0, status.CustomCount)
	assert.Equal(t, 1, status.TrancoCount)
}

func TestManager_ShippedCustomList_AllowsDiscussionDomains(t *testing.T) {
	customPath := filepath.Join("..", "..", "blocklists", "custom.txt")
	require.FileExists(t, customPath)

	cfg := &config.BlocklistConfig{CustomPath: customPath}
	m := NewManager(cfg, nil)
	require.NoError(t, m.LoadCustom())
	m.SetState(StateReady)

	assert.True(t, m.IsBlocked("facebook.com"))
	assert.True(t, m.IsBlocked("pinterest.com"))

	assert.False(t, m.IsBlocked("reddit.com"))
	assert.False(t, m.IsBlocked("www.reddit.com"))
}

func TestManager_NewManager_StateEmpty(t *testing.T) {
	cfg := &config.BlocklistConfig{}
	m := NewManager(cfg, nil)

	assert.Equal(t, StateEmpty, m.State(), "new manager should start in Empty state")
}

func TestManager_IsBlocked_ReturnsFalseInEmptyState(t *testing.T) {
	cfg := &config.BlocklistConfig{}
	m := NewManager(cfg, nil)

	assert.False(t, m.IsBlocked("anything.com"))
}

func TestManager_GetTrancoRank_ReturnsZeroInEmptyState(t *testing.T) {
	cfg := &config.BlocklistConfig{}
	m := NewManager(cfg, nil)

	assert.Equal(t, 0, m.GetTrancoRank("google.com"))
}

func TestManager_StartBackgroundLoad_TransitionsToReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("||bg-ads.test.com^\n"))
	}))
	defer server.Close()

	cfg := &config.BlocklistConfig{
		EasyList:    true,
		StevenBlack: false,
		Tranco:      false,
		CachePath:   t.TempDir(),
	}
	m := NewManager(cfg, nil)

	assert.Equal(t, StateEmpty, m.State())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	m.StartBackgroundLoad(ctx)

	assert.Equal(t, StateLoading, m.State())

	require.Eventually(t, func() bool {
		s := m.State()
		return s == StateReady || s == StateError
	}, 3*time.Second, 10*time.Millisecond, "state should transition to Ready or Error")
}

func TestManager_StartBackgroundLoad_MakesDomainsBlocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("||lazy-ads.example.com^\n||lazy-tracker.example.net^\n"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "easylist.txt")
	err := os.WriteFile(cachePath, []byte("||lazy-ads.example.com^\n||lazy-tracker.example.net^\n"), 0644)
	require.NoError(t, err)

	cfg := &config.BlocklistConfig{
		EasyList:     true,
		StevenBlack:  false,
		Tranco:       false,
		CachePath:    tmpDir,
		Mode:         "offline",
		RefreshHours: 24,
	}
	m := NewManager(cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assert.False(t, m.IsBlocked("lazy-ads.example.com"))

	m.StartBackgroundLoad(ctx)

	require.Eventually(t, func() bool {
		return m.State() == StateReady
	}, 3*time.Second, 10*time.Millisecond)

	assert.True(t, m.IsBlocked("lazy-ads.example.com"))
	assert.True(t, m.IsBlocked("lazy-tracker.example.net"))
	assert.False(t, m.IsBlocked("good.example.com"))
}

func TestManager_StartBackgroundLoad_OnlyLoadsOnce(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "easylist.txt")
	err := os.WriteFile(cachePath, []byte("||once.example.com^\n"), 0644)
	require.NoError(t, err)

	cfg := &config.BlocklistConfig{
		EasyList:     true,
		StevenBlack:  false,
		Tranco:       false,
		CachePath:    tmpDir,
		Mode:         "offline",
		RefreshHours: 24,
	}
	m := NewManager(cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	m.StartBackgroundLoad(ctx)
	m.StartBackgroundLoad(ctx)

	require.Eventually(t, func() bool {
		return m.State() == StateReady
	}, 3*time.Second, 10*time.Millisecond)

	assert.True(t, m.IsBlocked("once.example.com"))
}

func TestManager_Status_IncludesState(t *testing.T) {
	cfg := &config.BlocklistConfig{}
	m := NewManager(cfg, nil)

	status := m.Status()
	assert.Equal(t, StateEmpty, status.LoadState)
}

func TestManager_Load_SetsStateToReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("||sync.example.com^\n"))
	}))
	defer server.Close()

	cfg := &config.BlocklistConfig{
		EasyList:    true,
		StevenBlack: false,
		Tranco:      false,
		CachePath:   t.TempDir(),
	}
	m := NewManager(cfg, nil)

	ctx := context.Background()
	err := m.loadEasyListFrom(ctx, server.URL)
	require.NoError(t, err)

	cfg.Mode = "offline"
	cfg.RefreshHours = 24

	loadCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = m.Load(loadCtx)
	require.NoError(t, err)
	assert.Equal(t, StateReady, m.State())
}
