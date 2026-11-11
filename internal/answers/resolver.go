package answers

import (
	"context"
	"time"

	"github.com/adriangitvitz/kurai/internal/config"
)

type Cache interface {
	Get(userID, key string) ([]any, bool)
	Set(userID, key string, results []any)
}

type Resolver struct {
	cfg   *config.AnswersConfig
	cache Cache
	fx    *frankfurterClient
	om    *openMeteoClient

	now func() time.Time
}

func NewResolver(cfg *config.AnswersConfig, cache Cache) *Resolver {
	r := &Resolver{cfg: cfg, cache: cache, now: time.Now}
	if cfg.Currency.Enabled {
		r.fx = newFrankfurterClient(cfg.Currency.BaseURL, cfg.Currency.Timeout)
	}
	if cfg.Weather.Enabled {
		r.om = newOpenMeteoClient(cfg.Weather.GeocodingURL, cfg.Weather.ForecastURL, cfg.Weather.Timeout)
	}
	return r
}

func (r *Resolver) InfoboxEnabled() bool {
	return r.cfg.Infobox.Enabled
}

func (r *Resolver) ResolveCards(ctx context.Context, userID, query string) *Block {
	if r.cfg.Currency.Enabled {
		if trig := ParseCurrencyTrigger(query); trig != nil {
			if conv := r.resolveCurrencyCard(ctx, userID, trig); conv != nil {
				return &Block{Currency: conv}
			}
			return nil
		}
	}
	if r.cfg.Weather.Enabled {
		if place, ok := ParseWeatherTrigger(query); ok {
			if card := r.resolveWeatherCard(ctx, userID, place); card != nil {
				return &Block{Weather: card}
			}
		}
	}
	return nil
}
