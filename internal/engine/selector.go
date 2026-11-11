package engine

import (
	"sync"
)

type Category string

const (
	CategoryTech     Category = "tech"
	CategoryAcademic Category = "academic"
	CategoryNews     Category = "news"
	CategorySports   Category = "sports"
	CategoryImages   Category = "images"
	CategoryGeneral  Category = "general"
)

type Engine string

const (
	EngineGoogle           Engine = "google"
	EngineBing             Engine = "bing"
	EngineBrave            Engine = "brave"
	EngineDuckDuckGo       Engine = "duckduckgo"
	EngineGitHub           Engine = "github"
	EngineStackOverflow    Engine = "stackoverflow"
	EngineWikipedia        Engine = "wikipedia"
	EngineArxiv            Engine = "arxiv"
	EngineHackerNews       Engine = "hackernews"
	EngineMDN              Engine = "mdn"
	EngineNPM              Engine = "npm"
	EnginePyPI             Engine = "pypi"
	EngineGoogleNews       Engine = "google news"
	EngineBingNews         Engine = "bing news"
	EngineGoogleImages     Engine = "google images"
	EngineBingImages       Engine = "bing images"
	EngineUnsplash         Engine = "unsplash"
	EngineFlickr           Engine = "flickr"
	EngineWikimediaCommons Engine = "wikimedia commons"
)

type SelectorConfig struct {
	MinEngines       int                   `yaml:"min_engines"`
	MaxEngines       int                   `yaml:"max_engines"`
	CategoryMappings map[Category][]Engine `yaml:"category_mappings"`
	AlwaysInclude    []Engine              `yaml:"always_include"`
	NeverExclude     []Engine              `yaml:"never_exclude"`
}

func DefaultSelectorConfig() SelectorConfig {
	return SelectorConfig{
		MinEngines: 3,
		MaxEngines: 8,
		CategoryMappings: map[Category][]Engine{
			CategoryTech: {
				EngineGoogle, EngineBrave, EngineGitHub, EngineStackOverflow,
				EngineMDN, EngineNPM, EnginePyPI,
			},
			CategoryAcademic: {
				EngineGoogle, EngineBrave, EngineArxiv, EngineWikipedia,
			},
			CategoryNews: {
				EngineGoogle, EngineBrave, EngineBing, EngineHackerNews,
				EngineGoogleNews, EngineBingNews,
			},
			CategorySports: {
				EngineGoogle, EngineBrave, EngineBing,
			},
			CategoryImages: {
				EngineGoogleImages, EngineBingImages, EngineUnsplash,
				EngineFlickr, EngineWikimediaCommons,
			},
			CategoryGeneral: {
				EngineGoogle, EngineBrave, EngineBing, EngineDuckDuckGo,
				EngineWikipedia,
			},
		},
		AlwaysInclude: nil,
		NeverExclude:  []Engine{EngineGoogle, EngineBrave},
	}
}

type Selector struct {
	mu            sync.RWMutex
	config        SelectorConfig
	healthMonitor *HealthMonitor
}

func NewSelector(config SelectorConfig) *Selector {
	return &Selector{
		config:        config,
		healthMonitor: nil,
	}
}

func NewSelectorWithHealth(config SelectorConfig, healthMonitor *HealthMonitor) *Selector {
	return &Selector{
		config:        config,
		healthMonitor: healthMonitor,
	}
}

type SelectionResult struct {
	Engines         []Engine         `json:"engines"`
	Category        Category         `json:"category"`
	Reason          string           `json:"reason"`
	ExcludedEngines []ExcludedEngine `json:"excluded_engines,omitempty"`
}

type ExcludedEngine struct {
	Engine Engine `json:"engine"`
	Reason string `json:"reason"`
}

func (s *Selector) Select(category Category, override []Engine) SelectionResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := SelectionResult{
		Category: category,
	}

	if len(override) > 0 {
		result.Engines = s.validateEngines(override)
		result.Reason = "explicit override"
		return result
	}

	categoryEngines, ok := s.config.CategoryMappings[category]
	if !ok {
		categoryEngines = s.config.CategoryMappings[CategoryGeneral]
		result.Reason = "category fallback to general"
	} else {
		result.Reason = "category mapping"
	}

	engineSet := make(map[Engine]bool)
	var selectedEngines []Engine

	for _, e := range s.config.AlwaysInclude {
		if engineSet[e] {
			continue
		}
		if s.healthMonitor != nil && !s.healthMonitor.IsHealthy(string(e)) {
			result.ExcludedEngines = append(result.ExcludedEngines, ExcludedEngine{
				Engine: e,
				Reason: "unhealthy",
			})
			continue
		}
		selectedEngines = append(selectedEngines, e)
		engineSet[e] = true
	}

	for _, e := range categoryEngines {
		if engineSet[e] {
			continue
		}

		if s.healthMonitor != nil && !s.healthMonitor.IsHealthy(string(e)) {
			result.ExcludedEngines = append(result.ExcludedEngines, ExcludedEngine{
				Engine: e,
				Reason: "unhealthy",
			})
			continue
		}

		selectedEngines = append(selectedEngines, e)
		engineSet[e] = true

		if len(selectedEngines) >= s.config.MaxEngines {
			break
		}
	}

	if len(selectedEngines) < s.config.MinEngines {
		for _, e := range s.config.CategoryMappings[CategoryGeneral] {
			if engineSet[e] {
				continue
			}

			if s.healthMonitor != nil && !s.healthMonitor.IsHealthy(string(e)) {
				continue
			}

			selectedEngines = append(selectedEngines, e)
			engineSet[e] = true

			if len(selectedEngines) >= s.config.MinEngines {
				break
			}
		}
	}

	result.Engines = selectedEngines
	return result
}

func (s *Selector) SelectForIntent(category string, isDoc, isError, isNews bool) SelectionResult {
	var cat Category

	switch category {
	case "tech":
		cat = CategoryTech
	case "academic":
		cat = CategoryAcademic
	case "news":
		cat = CategoryNews
	case "sports":
		cat = CategorySports
	default:
		cat = CategoryGeneral
	}

	if isNews {
		cat = CategoryNews
	}

	result := s.Select(cat, nil)

	if isDoc {
		result = s.boostEngines(result, []Engine{EngineMDN, EngineStackOverflow})
	}

	if isError {
		result = s.boostEngines(result, []Engine{EngineStackOverflow, EngineGitHub})
	}

	return result
}

func (s *Selector) boostEngines(result SelectionResult, toBoost []Engine) SelectionResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	engineSet := make(map[Engine]bool)
	for _, e := range result.Engines {
		engineSet[e] = true
	}

	for _, e := range toBoost {
		if !engineSet[e] {
			if s.healthMonitor != nil && !s.healthMonitor.IsHealthy(string(e)) {
				continue
			}

			result.Engines = append(result.Engines, e)
			engineSet[e] = true

			if len(result.Engines) > s.config.MaxEngines {
				for i := len(result.Engines) - 2; i >= 0; i-- {
					if !s.isNeverExclude(result.Engines[i]) {
						result.Engines = append(result.Engines[:i], result.Engines[i+1:]...)
						break
					}
				}
			}
		}
	}

	return result
}

func (s *Selector) validateEngines(engines []Engine) []Engine {
	var valid []Engine

	for _, e := range engines {
		if s.healthMonitor != nil && !s.healthMonitor.IsHealthy(string(e)) {
			continue
		}
		valid = append(valid, e)
	}

	engineSet := make(map[Engine]bool)
	for _, e := range valid {
		engineSet[e] = true
	}

	for _, e := range s.config.NeverExclude {
		if !engineSet[e] {
			valid = append(valid, e)
		}
	}

	return valid
}

func (s *Selector) isNeverExclude(e Engine) bool {
	for _, ne := range s.config.NeverExclude {
		if ne == e {
			return true
		}
	}
	return false
}

func (s *Selector) GetCategoryEngines(category Category) []Engine {
	s.mu.RLock()
	defer s.mu.RUnlock()

	engines, ok := s.config.CategoryMappings[category]
	if !ok {
		return s.config.CategoryMappings[CategoryGeneral]
	}
	return engines
}

func (s *Selector) SetConfig(config SelectorConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

func (s *Selector) SetHealthMonitor(monitor *HealthMonitor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthMonitor = monitor
}

func (s *Selector) AddCategoryMapping(category Category, engines []Engine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.CategoryMappings[category] = engines
}

func EngineListToStrings(engines []Engine) []string {
	result := make([]string, len(engines))
	for i, e := range engines {
		result[i] = string(e)
	}
	return result
}

func StringsToEngineList(strings []string) []Engine {
	result := make([]Engine, len(strings))
	for i, s := range strings {
		result[i] = Engine(s)
	}
	return result
}

func ParseCategory(s string) Category {
	switch s {
	case "tech", "technical", "programming":
		return CategoryTech
	case "academic", "research", "science":
		return CategoryAcademic
	case "news", "current", "events":
		return CategoryNews
	case "sports", "sport":
		return CategorySports
	case "images", "image", "pictures":
		return CategoryImages
	default:
		return CategoryGeneral
	}
}
