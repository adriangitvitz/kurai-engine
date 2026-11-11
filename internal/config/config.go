package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Tavily          TavilyConfig          `yaml:"tavily"`
	SearXNG         SearXNGConfig         `yaml:"searxng"`
	Server          ServerConfig          `yaml:"server"`
	Ranking         RankingConfig         `yaml:"ranking"`
	Blocklists      BlocklistConfig       `yaml:"blocklists"`
	Probe           ProbeConfig           `yaml:"probe"`
	Privacy         PrivacyConfig         `yaml:"privacy"`
	EngineSelection EngineSelectionConfig `yaml:"engine_selection"`
	Images          ImagesConfig          `yaml:"images"`
	News            NewsConfig            `yaml:"news"`
	QueryAnalysis   QueryAnalysisConfig   `yaml:"query_analysis"`
	Diversity       DiversityConfig       `yaml:"diversity"`
	DataPaths       DataPathsConfig       `yaml:"data_paths"`
	Auth            AuthConfig            `yaml:"auth"`
	Extract         ExtractConfig         `yaml:"extract"`
	MCP             MCPConfig             `yaml:"mcp"`
	Reranker        RerankerConfig        `yaml:"reranker"`
	Answers         AnswersConfig         `yaml:"answers"`
	Bookmarks       BookmarksConfig       `yaml:"bookmarks"`
}

type BookmarksConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type AnswersConfig struct {
	Enabled  bool                 `yaml:"enabled"`
	Currency CurrencyAnswerConfig `yaml:"currency"`
	Weather  WeatherAnswerConfig  `yaml:"weather"`
	Infobox  InfoboxAnswerConfig  `yaml:"infobox"`
}

type CurrencyAnswerConfig struct {
	Enabled bool          `yaml:"enabled"`
	BaseURL string        `yaml:"base_url"`
	Timeout time.Duration `yaml:"timeout"`
}

type WeatherAnswerConfig struct {
	Enabled      bool          `yaml:"enabled"`
	GeocodingURL string        `yaml:"geocoding_url"`
	ForecastURL  string        `yaml:"forecast_url"`
	Timeout      time.Duration `yaml:"timeout"`
}

type InfoboxAnswerConfig struct {
	Enabled bool `yaml:"enabled"`
}

type RerankerConfig struct {
	Enabled    bool    `yaml:"enabled"`
	Provider   string  `yaml:"provider"`
	URL        string  `yaml:"url"`
	APIKeyEnv  string  `yaml:"api_key_env"`
	TimeoutMs  int     `yaml:"timeout_ms"`
	TopN       int     `yaml:"top_n"`
	BlendAlpha float64 `yaml:"blend_alpha"`
	_          time.Duration
}

type MCPConfig struct {
	Enabled        bool          `yaml:"enabled"`
	KeepAlive      time.Duration `yaml:"keepalive"`
	SessionTimeout time.Duration `yaml:"session_timeout"`
}

type ExtractConfig struct {
	Enabled              bool          `yaml:"enabled"`
	Timeout              time.Duration `yaml:"timeout"`
	MaxBytes             int64         `yaml:"max_bytes"`
	MaxRedirects         int           `yaml:"max_redirects"`
	UserAgent            string        `yaml:"user_agent"`
	AllowPrivateNetworks bool          `yaml:"allow_private_networks"`
	MaxConcurrent        int           `yaml:"max_concurrent"`
	FusedBudget          time.Duration `yaml:"fused_budget"`
}

type AuthConfig struct {
	Enabled         bool          `yaml:"enabled"`
	ServerSecret    string        `yaml:"server_secret"`
	SessionTTL      time.Duration `yaml:"session_ttl"`
	UsersFile       string        `yaml:"users_file"`
	MaxPerUser      int           `yaml:"max_cache_per_user"`
	RateLimit       int           `yaml:"rate_limit"`
	BearerRateLimit int           `yaml:"bearer_rate_limit"`
	CookieSecure    bool          `yaml:"cookie_secure"`
}

type PrivacyConfig struct {
	DoH DoHConfig `yaml:"doh"`
}

type DoHConfig struct {
	Enabled     bool   `yaml:"enabled"`
	ResolverURL string `yaml:"resolver_url"`
}

type SearXNGConfig struct {
	URL     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`
}

type TavilyConfig struct {
	URL         string        `yaml:"url"`
	Enabled     bool          `yaml:"enabled"`
	APIKeyEnv   string        `yaml:"api_key_env"`
	SearchDepth string        `yaml:"search_depth"`
	MaxResults  int           `yaml:"max_results"`
	Timeout     time.Duration `yaml:"timeout"`
}

type ServerConfig struct {
	Port          int           `yaml:"port"`
	ReadTimeout   time.Duration `yaml:"read_timeout"`
	WriteTimeout  time.Duration `yaml:"write_timeout"`
	CacheTTL      time.Duration `yaml:"cache_ttl"`
	PageSize      int           `yaml:"page_size"`
	PrefetchPages int           `yaml:"prefetch_pages"`
}

type RankingConfig struct {
	Weights        WeightConfig         `yaml:"weights"`
	Penalties      PenaltyConfig        `yaml:"penalties"`
	Authority      AuthorityConfig      `yaml:"authority"`
	Snippet        SnippetConfig        `yaml:"snippet"`
	URLStructure   URLStructureConfig   `yaml:"url_structure"`
	RRF            RRFConfig            `yaml:"rrf"`
	Alignment      AlignmentConfig      `yaml:"alignment"`
	RelevanceFloor RelevanceFloorConfig `yaml:"relevance_floor"`
}

type RelevanceFloorConfig struct {
	Enabled     *bool   `yaml:"enabled"`
	MinCoverage float64 `yaml:"min_coverage"`
}

type RRFConfig struct {
	Enabled       bool               `yaml:"enabled"`
	K             float64            `yaml:"k"`
	DefaultRank   int                `yaml:"default_rank"`
	SignalWeights map[string]float64 `yaml:"signal_weights"`
}

type AlignmentConfig struct {
	MaxBoost        float64 `yaml:"max_boost"`
	MismatchPenalty float64 `yaml:"mismatch_penalty"`
	MinConfidence   float64 `yaml:"min_confidence"`
}

type WeightConfig struct {
	TitleMatch      int `yaml:"title_match"`
	URLMatch        int `yaml:"url_match"`
	EngineAgreement int `yaml:"engine_agreement"`
	TrancoTop10k    int `yaml:"tranco_top_10k"`
	TrancoTop100k   int `yaml:"tranco_top_100k"`
	TrancoTop1M     int `yaml:"tranco_top_1m"`
	Freshness       int `yaml:"freshness"`
}

type PenaltyConfig struct {
	Blocklist       int `yaml:"blocklist"`
	ContentFarm     int `yaml:"content_farm"`
	ExcessiveParams int `yaml:"excessive_params"`
	ParamThreshold  int `yaml:"param_threshold"`
}

type AuthorityConfig struct {
	OfficialDocs   int `yaml:"official_docs"`
	CodeRepository int `yaml:"code_repository"`
	DeveloperQA    int `yaml:"developer_qa"`
	QualityBlog    int `yaml:"quality_blog"`

	PaperArchive      int `yaml:"paper_archive"`
	AcademicPublisher int `yaml:"academic_publisher"`
	Wikipedia         int `yaml:"wikipedia"`

	MajorNews   int `yaml:"major_news"`
	QualityNews int `yaml:"quality_news"`

	QueryDomainMatch int `yaml:"query_domain_match"`
}

type SnippetConfig struct {
	PartialMatchMax    int `yaml:"partial_match_max"`
	TermFrequencyMax   int `yaml:"term_frequency_max"`
	ProximityBonus     int `yaml:"proximity_bonus"`
	CodeDetectionBonus int `yaml:"code_detection_bonus"`
	LowQualityPenalty  int `yaml:"low_quality_penalty"`
	HighQualityBonus   int `yaml:"high_quality_bonus"`
}

type URLStructureConfig struct {
	DocumentationPath int `yaml:"documentation_path"`
	GitHubCode        int `yaml:"github_code"`
	GitHubIssue       int `yaml:"github_issue"`
	AggregatorPenalty int `yaml:"aggregator_penalty"`
}

type BlocklistConfig struct {
	EasyList        bool   `yaml:"easylist"`
	StevenBlack     bool   `yaml:"stevenblack"`
	Tranco          bool   `yaml:"tranco"`
	CustomPath      string `yaml:"custom"`
	ContentFarmPath string `yaml:"content_farms"`
	CachePath       string `yaml:"cache_path"`
	Mode            string `yaml:"mode"`
	RefreshHours    int    `yaml:"refresh_hours"`
}

type ProbeConfig struct {
	Enabled              bool          `yaml:"enabled"`
	TopN                 int           `yaml:"top_n"`
	Timeout              time.Duration `yaml:"timeout"`
	Concurrency          int           `yaml:"concurrency"`
	CacheTTL             time.Duration `yaml:"cache_ttl"`
	DropUnreachable      bool          `yaml:"drop_unreachable"`
	AllowPrivateNetworks bool          `yaml:"allow_private_networks"`
}

type EngineSelectionConfig struct {
	Enabled          bool                `yaml:"enabled"`
	MinEngines       int                 `yaml:"min_engines"`
	MaxEngines       int                 `yaml:"max_engines"`
	CategoryMappings map[string][]string `yaml:"category_mappings"`
	AlwaysInclude    []string            `yaml:"always_include"`
	HealthMonitoring EngineHealthConfig  `yaml:"health_monitoring"`
}

type EngineHealthConfig struct {
	Enabled          bool          `yaml:"enabled"`
	FailureThreshold int           `yaml:"failure_threshold"`
	SuccessThreshold int           `yaml:"success_threshold"`
	LatencyWindow    int           `yaml:"latency_window"`
	MaxLatency       time.Duration `yaml:"max_latency"`
	RecoveryDelay    time.Duration `yaml:"recovery_delay"`
}

type ImagesConfig struct {
	Enabled           bool          `yaml:"enabled"`
	DefaultSafeSearch int           `yaml:"default_safesearch"`
	PageSize          int           `yaml:"page_size"`
	CacheTTL          time.Duration `yaml:"cache_ttl"`
	RelevanceWeight   float64       `yaml:"relevance_weight"`
	QualityWeight     float64       `yaml:"quality_weight"`
	AuthorityWeight   float64       `yaml:"authority_weight"`
}

type NewsConfig struct {
	Enabled           bool          `yaml:"enabled"`
	DefaultTimeRange  string        `yaml:"default_time_range"`
	EnableClustering  bool          `yaml:"enable_clustering"`
	EnableTrending    bool          `yaml:"enable_trending"`
	PageSize          int           `yaml:"page_size"`
	CacheTTL          time.Duration `yaml:"cache_ttl"`
	TrendingCacheTTL  time.Duration `yaml:"trending_cache_ttl"`
	RecencyWeight     float64       `yaml:"recency_weight"`
	CredibilityWeight float64       `yaml:"credibility_weight"`
	RelevanceWeight   float64       `yaml:"relevance_weight"`
	ClusterSimilarity float64       `yaml:"cluster_similarity"`
}

type QueryAnalysisConfig struct {
	EntityRecognition bool `yaml:"entity_recognition"`
	QueryTypeClassify bool `yaml:"query_type_classify"`
	MultiIntent       bool `yaml:"multi_intent"`
	SynonymExpansion  bool `yaml:"synonym_expansion"`
	AcronymExpansion  bool `yaml:"acronym_expansion"`
	MaxSynonymExpand  int  `yaml:"max_synonym_expand"`
}

type DiversityConfig struct {
	Enabled          bool     `yaml:"enabled"`
	MaxPerDomain     int      `yaml:"max_per_domain"`
	PenaltyPerExtra  float64  `yaml:"penalty_per_extra"`
	ExcludedDomains  []string `yaml:"excluded_domains"`
	EnableForNews    bool     `yaml:"enable_for_news"`
	NewsMaxPerSource int      `yaml:"news_max_per_source"`
}

type DataPathsConfig struct {
	EntitiesFile string `yaml:"entities_file"`
	SynonymsFile string `yaml:"synonyms_file"`
	AcronymsFile string `yaml:"acronyms_file"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	setDefaults(cfg)

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("reading config: %w", err)
			}
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parsing config: %w", err)
			}
		}
	}

	applyEnvOverrides(cfg)

	return cfg, nil
}

func setDefaults(cfg *Config) {
	cfg.Tavily.Enabled = false
	cfg.Tavily.URL = "https://api.tavily.com"
	cfg.Tavily.APIKeyEnv = "TAVILY_API_KEY"
	cfg.Tavily.SearchDepth = "basic"
	cfg.Tavily.MaxResults = 5
	cfg.Tavily.Timeout = 8 * time.Second

	cfg.SearXNG.URL = "http://localhost:8888"
	cfg.SearXNG.Timeout = 5 * time.Second

	cfg.Server.Port = 8080
	cfg.Server.ReadTimeout = 30 * time.Second
	cfg.Server.WriteTimeout = 30 * time.Second
	cfg.Server.CacheTTL = 5 * time.Minute
	cfg.Server.PageSize = 20
	cfg.Server.PrefetchPages = 3

	cfg.Ranking.Weights.TitleMatch = 20
	cfg.Ranking.Weights.URLMatch = 10
	cfg.Ranking.Weights.EngineAgreement = 5
	cfg.Ranking.Weights.TrancoTop10k = 15
	cfg.Ranking.Weights.TrancoTop100k = 10
	cfg.Ranking.Weights.TrancoTop1M = 5
	cfg.Ranking.Weights.Freshness = 5

	cfg.Ranking.Penalties.Blocklist = -50
	cfg.Ranking.Penalties.ContentFarm = -30
	cfg.Ranking.Penalties.ExcessiveParams = -10
	cfg.Ranking.Penalties.ParamThreshold = 5

	cfg.Ranking.Authority.OfficialDocs = 25
	cfg.Ranking.Authority.CodeRepository = 15
	cfg.Ranking.Authority.DeveloperQA = 12
	cfg.Ranking.Authority.QualityBlog = 8
	cfg.Ranking.Authority.PaperArchive = 25
	cfg.Ranking.Authority.AcademicPublisher = 20
	cfg.Ranking.Authority.Wikipedia = 15
	cfg.Ranking.Authority.MajorNews = 15
	cfg.Ranking.Authority.QualityNews = 12
	cfg.Ranking.Authority.QueryDomainMatch = 30

	cfg.Ranking.Snippet.PartialMatchMax = 20
	cfg.Ranking.Snippet.TermFrequencyMax = 15
	cfg.Ranking.Snippet.ProximityBonus = 5
	cfg.Ranking.Snippet.CodeDetectionBonus = 10
	cfg.Ranking.Snippet.LowQualityPenalty = -10
	cfg.Ranking.Snippet.HighQualityBonus = 8

	cfg.Ranking.URLStructure.DocumentationPath = 10
	cfg.Ranking.URLStructure.GitHubCode = 8
	cfg.Ranking.URLStructure.GitHubIssue = 6
	cfg.Ranking.URLStructure.AggregatorPenalty = -15

	cfg.Ranking.RRF.Enabled = true
	cfg.Ranking.RRF.K = 60.0
	cfg.Ranking.RRF.DefaultRank = 1000
	cfg.Ranking.RRF.SignalWeights = map[string]float64{
		"searxng_position": 1.0,
		"text_relevance":   1.2,
		"domain_alignment": 0.8,
		"tranco_authority": 0.5,
		"engine_agreement": 0.6,
		"freshness":        0.4,
	}

	relevanceFloorOn := true
	cfg.Ranking.RelevanceFloor.Enabled = &relevanceFloorOn
	cfg.Ranking.RelevanceFloor.MinCoverage = 0

	cfg.Ranking.Alignment.MaxBoost = 0.5
	cfg.Ranking.Alignment.MismatchPenalty = 0.2
	cfg.Ranking.Alignment.MinConfidence = 0.3

	cfg.Blocklists.EasyList = true
	cfg.Blocklists.StevenBlack = true
	cfg.Blocklists.Tranco = true
	cfg.Blocklists.CustomPath = "./blocklists/custom.txt"
	cfg.Blocklists.ContentFarmPath = "./blocklists/content-farms.txt"
	cfg.Blocklists.CachePath = "./blocklists/cache"
	cfg.Blocklists.Mode = "cache-first"
	cfg.Blocklists.RefreshHours = 24

	cfg.Probe.Enabled = true
	cfg.Probe.TopN = 20
	cfg.Probe.Timeout = 2 * time.Second
	cfg.Probe.Concurrency = 12
	cfg.Probe.CacheTTL = 15 * time.Minute
	cfg.Probe.DropUnreachable = true

	cfg.Privacy.DoH.Enabled = true
	cfg.Privacy.DoH.ResolverURL = "https://1.1.1.1/dns-query"

	cfg.EngineSelection.Enabled = true
	cfg.EngineSelection.MinEngines = 3
	cfg.EngineSelection.MaxEngines = 8
	cfg.EngineSelection.AlwaysInclude = nil
	cfg.EngineSelection.CategoryMappings = map[string][]string{
		"tech":     {"google", "brave", "github", "stackoverflow", "mdn"},
		"academic": {"google", "brave", "arxiv", "wikipedia"},
		"news":     {"google", "brave", "bing", "hackernews"},
		"sports":   {"google", "brave", "bing"},
		"images":   {"google images", "bing images", "unsplash"},
		"general":  {"google", "brave", "bing", "duckduckgo", "wikipedia"},
	}
	cfg.EngineSelection.HealthMonitoring.Enabled = true
	cfg.EngineSelection.HealthMonitoring.FailureThreshold = 3
	cfg.EngineSelection.HealthMonitoring.SuccessThreshold = 2
	cfg.EngineSelection.HealthMonitoring.LatencyWindow = 10
	cfg.EngineSelection.HealthMonitoring.MaxLatency = 10 * time.Second
	cfg.EngineSelection.HealthMonitoring.RecoveryDelay = 60 * time.Second

	cfg.Images.Enabled = true
	cfg.Images.DefaultSafeSearch = 1
	cfg.Images.PageSize = 30
	cfg.Images.CacheTTL = 1 * time.Hour
	cfg.Images.RelevanceWeight = 0.5
	cfg.Images.QualityWeight = 0.3
	cfg.Images.AuthorityWeight = 0.2

	cfg.News.Enabled = true
	cfg.News.DefaultTimeRange = "week"
	cfg.News.EnableClustering = true
	cfg.News.EnableTrending = true
	cfg.News.PageSize = 20
	cfg.News.CacheTTL = 5 * time.Minute
	cfg.News.TrendingCacheTTL = 2 * time.Minute
	cfg.News.RecencyWeight = 0.4
	cfg.News.CredibilityWeight = 0.35
	cfg.News.RelevanceWeight = 0.25
	cfg.News.ClusterSimilarity = 0.6

	cfg.QueryAnalysis.EntityRecognition = true
	cfg.QueryAnalysis.QueryTypeClassify = true
	cfg.QueryAnalysis.MultiIntent = true
	cfg.QueryAnalysis.SynonymExpansion = true
	cfg.QueryAnalysis.AcronymExpansion = true
	cfg.QueryAnalysis.MaxSynonymExpand = 3

	cfg.Diversity.Enabled = true
	cfg.Diversity.MaxPerDomain = 3
	cfg.Diversity.PenaltyPerExtra = 0.15
	cfg.Diversity.ExcludedDomains = []string{"github.com", "stackoverflow.com"}
	cfg.Diversity.EnableForNews = true
	cfg.Diversity.NewsMaxPerSource = 2

	cfg.DataPaths.EntitiesFile = "./data/entities.yaml"
	cfg.DataPaths.SynonymsFile = "./data/synonyms.yaml"
	cfg.DataPaths.AcronymsFile = "./data/acronyms.yaml"

	cfg.Auth.Enabled = false
	cfg.Auth.SessionTTL = 8 * time.Hour
	cfg.Auth.UsersFile = "./users.yaml"
	cfg.Auth.MaxPerUser = 1000
	cfg.Auth.RateLimit = 5
	cfg.Auth.BearerRateLimit = 60
	cfg.Auth.CookieSecure = true

	cfg.Extract.Enabled = true
	cfg.Extract.Timeout = 10 * time.Second
	cfg.Extract.MaxBytes = 5 << 20
	cfg.Extract.MaxRedirects = 5
	cfg.Extract.UserAgent = "kurai-extract/1.0"
	cfg.Extract.AllowPrivateNetworks = false
	cfg.Extract.MaxConcurrent = 16
	cfg.Extract.FusedBudget = 8 * time.Second

	cfg.MCP.Enabled = true
	cfg.MCP.KeepAlive = 30 * time.Second
	cfg.MCP.SessionTimeout = 30 * time.Minute

	cfg.Reranker.Enabled = false
	cfg.Reranker.Provider = "noop"
	cfg.Reranker.TimeoutMs = 200
	cfg.Reranker.TopN = 30
	cfg.Reranker.BlendAlpha = 0.6

	cfg.Answers.Enabled = true
	cfg.Answers.Currency.Enabled = true
	cfg.Answers.Currency.BaseURL = "https://api.frankfurter.dev"
	cfg.Answers.Currency.Timeout = 3 * time.Second
	cfg.Answers.Weather.Enabled = true
	cfg.Answers.Weather.GeocodingURL = "https://geocoding-api.open-meteo.com"
	cfg.Answers.Weather.ForecastURL = "https://api.open-meteo.com"
	cfg.Answers.Weather.Timeout = 3 * time.Second
	cfg.Answers.Infobox.Enabled = true

	cfg.Bookmarks.Enabled = true
	cfg.Bookmarks.Path = "/app/data/bookmarks.db"
}

func applyEnvOverrides(cfg *Config) {
	if url := os.Getenv("KURAI_SEARXNG_URL"); url != "" {
		cfg.SearXNG.URL = url
	}

	if port := os.Getenv("KURAI_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}

	if enabled := os.Getenv("KURAI_PROBE_ENABLED"); enabled != "" {
		cfg.Probe.Enabled = enabled == "true" || enabled == "1"
	}

	if drop := os.Getenv("KURAI_PROBE_DROP_UNREACHABLE"); drop != "" {
		cfg.Probe.DropUnreachable = drop == "true" || drop == "1"
	}

	if custom := os.Getenv("KURAI_BLOCKLIST_CUSTOM"); custom != "" {
		cfg.Blocklists.CustomPath = custom
	}

	if doh := os.Getenv("KURAI_DOH_ENABLED"); doh != "" {
		cfg.Privacy.DoH.Enabled = doh == "true" || doh == "1"
	}

	if dohURL := os.Getenv("KURAI_DOH_RESOLVER"); dohURL != "" {
		cfg.Privacy.DoH.ResolverURL = dohURL
	}

	if cachePath := os.Getenv("KURAI_BLOCKLIST_CACHE_PATH"); cachePath != "" {
		cfg.Blocklists.CachePath = cachePath
	}

	if tav := os.Getenv("KURAI_TAVILY_ENABLED"); tav != "" {
		cfg.Tavily.Enabled = tav == "true" || tav == "1"
	}

	if ans := os.Getenv("KURAI_ANSWERS_ENABLED"); ans != "" {
		cfg.Answers.Enabled = ans == "true" || ans == "1"
	}

	if bm := os.Getenv("KURAI_BOOKMARKS_ENABLED"); bm != "" {
		cfg.Bookmarks.Enabled = bm == "true" || bm == "1"
	}
	if path := os.Getenv("KURAI_BOOKMARKS_PATH"); path != "" {
		cfg.Bookmarks.Path = path
	}

	if rrf := os.Getenv("KURAI_RRF_ENABLED"); rrf != "" {
		cfg.Ranking.RRF.Enabled = rrf == "true" || rrf == "1"
	}

	if rrfK := os.Getenv("KURAI_RRF_K"); rrfK != "" {
		if k, err := strconv.ParseFloat(rrfK, 64); err == nil {
			cfg.Ranking.RRF.K = k
		}
	}

	if authEnabled := os.Getenv("KURAI_AUTH_ENABLED"); authEnabled != "" {
		cfg.Auth.Enabled = authEnabled == "true" || authEnabled == "1"
	}
	if secret := os.Getenv("KURAI_AUTH_SECRET"); secret != "" {
		cfg.Auth.ServerSecret = secret
	}
	if usersFile := os.Getenv("KURAI_AUTH_USERS_FILE"); usersFile != "" {
		cfg.Auth.UsersFile = usersFile
	}
	if cookieSecure := os.Getenv("KURAI_COOKIE_SECURE"); cookieSecure != "" {
		cfg.Auth.CookieSecure = cookieSecure == "true" || cookieSecure == "1"
	}
}
