package config

import (
	"testing"
)

func TestLoad_ShippedConfigAnswersSection(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("shipped config.yaml failed to load: %v", err)
	}
	t.Logf("answers=%+v", cfg.Answers)
	if !cfg.Answers.Enabled || !cfg.Answers.Currency.Enabled || !cfg.Answers.Weather.Enabled || !cfg.Answers.Infobox.Enabled {
		t.Fatalf("answers pipeline not fully enabled: %+v", cfg.Answers)
	}
	if cfg.Answers.Currency.BaseURL == "" || cfg.Answers.Weather.GeocodingURL == "" {
		t.Fatalf("fetcher base URLs missing: %+v", cfg.Answers)
	}
}
