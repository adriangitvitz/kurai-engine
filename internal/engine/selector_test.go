package engine

import (
	"testing"
)

func TestSelector_Select_ByCategory(t *testing.T) {
	config := DefaultSelectorConfig()
	selector := NewSelector(config)

	tests := []struct {
		name        string
		category    Category
		wantEngines []Engine
		wantMinLen  int
		wantGoogle  bool
	}{
		{
			name:       "tech category",
			category:   CategoryTech,
			wantMinLen: 3,
			wantGoogle: true,
		},
		{
			name:       "academic category",
			category:   CategoryAcademic,
			wantMinLen: 3,
			wantGoogle: true,
		},
		{
			name:       "news category",
			category:   CategoryNews,
			wantMinLen: 3,
			wantGoogle: true,
		},
		{
			name:       "general category",
			category:   CategoryGeneral,
			wantMinLen: 3,
			wantGoogle: true,
		},
		{
			name:       "images category",
			category:   CategoryImages,
			wantMinLen: 3,
			wantGoogle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selector.Select(tt.category, nil)
			if len(result.Engines) < tt.wantMinLen {
				t.Errorf("Select(%v) got %d engines, want at least %d",
					tt.category, len(result.Engines), tt.wantMinLen)
			}

			hasGoogle := false
			for _, e := range result.Engines {
				if e == EngineGoogle {
					hasGoogle = true
					break
				}
			}
			if hasGoogle != tt.wantGoogle {
				t.Errorf("Select(%v) google present = %v, want %v (engines: %v)",
					tt.category, hasGoogle, tt.wantGoogle, result.Engines)
			}
		})
	}
}

func TestSelector_Select_WithOverride(t *testing.T) {
	config := DefaultSelectorConfig()
	selector := NewSelector(config)

	override := []Engine{EngineBing, EngineBrave}
	result := selector.Select(CategoryGeneral, override)

	if result.Reason != "explicit override" {
		t.Errorf("Select() with override got reason %q, want %q",
			result.Reason, "explicit override")
	}

	found := false
	for _, e := range result.Engines {
		if e == EngineGoogle || e == EngineBrave {
			found = true
			break
		}
	}
	if !found {
		t.Error("Select() with override missing never-exclude engines")
	}
}

func TestSelector_SelectForIntent(t *testing.T) {
	config := DefaultSelectorConfig()
	selector := NewSelector(config)

	tests := []struct {
		name     string
		category string
		isDoc    bool
		isError  bool
		isNews   bool
	}{
		{"tech docs", "tech", true, false, false},
		{"tech error", "tech", false, true, false},
		{"news override", "tech", false, false, true},
		{"general", "general", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selector.SelectForIntent(tt.category, tt.isDoc, tt.isError, tt.isNews)
			if len(result.Engines) == 0 {
				t.Error("SelectForIntent() returned no engines")
			}
		})
	}
}

func TestSelector_GetCategoryEngines(t *testing.T) {
	config := DefaultSelectorConfig()
	selector := NewSelector(config)

	tests := []struct {
		category Category
		wantMin  int
	}{
		{CategoryTech, 3},
		{CategoryAcademic, 3},
		{CategoryNews, 3},
		{CategoryGeneral, 3},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			engines := selector.GetCategoryEngines(tt.category)
			if len(engines) < tt.wantMin {
				t.Errorf("GetCategoryEngines(%v) got %d engines, want at least %d",
					tt.category, len(engines), tt.wantMin)
			}
		})
	}
}

func TestSelector_AddCategoryMapping(t *testing.T) {
	config := DefaultSelectorConfig()
	selector := NewSelector(config)

	customEngines := []Engine{EngineBing, EngineBrave}
	selector.AddCategoryMapping("custom", customEngines)

	engines := selector.GetCategoryEngines("custom")
	if len(engines) != 2 {
		t.Errorf("GetCategoryEngines(custom) got %d engines, want 2", len(engines))
	}
}

func TestSelector_WithHealthMonitor(t *testing.T) {
	config := DefaultSelectorConfig()
	healthConfig := DefaultHealthConfig()
	healthMonitor := NewHealthMonitor(healthConfig)

	healthMonitor.RegisterEngine("google")
	healthMonitor.RegisterEngine("bing")

	healthMonitor.RecordFailure("bing", nil)
	healthMonitor.RecordFailure("bing", nil)
	healthMonitor.RecordFailure("bing", nil)

	selector := NewSelectorWithHealth(config, healthMonitor)

	result := selector.Select(CategoryGeneral, nil)

	for _, e := range result.Engines {
		if e == EngineBing {
			t.Errorf("unhealthy bing was selected: %v", result.Engines)
		}
	}
	found := false
	for _, ex := range result.ExcludedEngines {
		if ex.Engine == EngineBing && ex.Reason == "unhealthy" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("bing excluded but not reported as unhealthy: %v", result.ExcludedEngines)
	}
}

func TestSelector_AlwaysInclude_RespectsHealth(t *testing.T) {
	config := DefaultSelectorConfig()
	config.AlwaysInclude = []Engine{EngineGoogle}

	healthMonitor := NewHealthMonitor(DefaultHealthConfig())
	healthMonitor.RegisterEngine("google")
	for range 3 {
		healthMonitor.RecordFailure("google", nil)
	}

	selector := NewSelectorWithHealth(config, healthMonitor)
	result := selector.Select(CategoryGeneral, nil)

	for _, e := range result.Engines {
		if e == EngineGoogle {
			t.Fatalf("unhealthy always-include engine was selected: %v", result.Engines)
		}
	}
	found := false
	for _, ex := range result.ExcludedEngines {
		if ex.Engine == EngineGoogle && ex.Reason == "unhealthy" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("google not reported as excluded-unhealthy: %v", result.ExcludedEngines)
	}
}

func TestParseCategory(t *testing.T) {
	tests := []struct {
		input string
		want  Category
	}{
		{"tech", CategoryTech},
		{"technical", CategoryTech},
		{"programming", CategoryTech},
		{"academic", CategoryAcademic},
		{"research", CategoryAcademic},
		{"news", CategoryNews},
		{"sports", CategorySports},
		{"images", CategoryImages},
		{"unknown", CategoryGeneral},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseCategory(tt.input); got != tt.want {
				t.Errorf("ParseCategory(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEngineListConversions(t *testing.T) {
	engines := []Engine{EngineGoogle, EngineBing, EngineBrave}
	strings := EngineListToStrings(engines)

	if len(strings) != 3 {
		t.Fatalf("EngineListToStrings() got %d, want 3", len(strings))
	}

	back := StringsToEngineList(strings)
	if len(back) != 3 {
		t.Fatalf("StringsToEngineList() got %d, want 3", len(back))
	}

	for i, e := range back {
		if e != engines[i] {
			t.Errorf("Engine %d: got %v, want %v", i, e, engines[i])
		}
	}
}

func TestSelector_MaxEnginesLimit(t *testing.T) {
	config := DefaultSelectorConfig()
	config.MaxEngines = 3
	selector := NewSelector(config)

	result := selector.Select(CategoryGeneral, nil)

	if len(result.Engines) > config.MaxEngines {
		t.Errorf("Select() returned %d engines, max is %d",
			len(result.Engines), config.MaxEngines)
	}
}

func TestSelector_MinEnginesGuarantee(t *testing.T) {
	config := DefaultSelectorConfig()
	config.MinEngines = 3
	config.CategoryMappings["sparse"] = []Engine{EngineBing}
	selector := NewSelector(config)

	result := selector.Select("sparse", nil)

	if len(result.Engines) < config.MinEngines {
		t.Errorf("Select() returned %d engines, min is %d",
			len(result.Engines), config.MinEngines)
	}
}
