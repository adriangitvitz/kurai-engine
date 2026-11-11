package engine

import (
	"sync"
	"time"
)

type HealthConfig struct {
	FailureThreshold int           `yaml:"failure_threshold"`
	SuccessThreshold int           `yaml:"success_threshold"`
	LatencyWindow    int           `yaml:"latency_window"`
	MaxLatency       time.Duration `yaml:"max_latency"`
	CheckInterval    time.Duration `yaml:"check_interval"`
	RecoveryDelay    time.Duration `yaml:"recovery_delay"`
	Enabled          bool          `yaml:"enabled"`
}

func DefaultHealthConfig() HealthConfig {
	return HealthConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		LatencyWindow:    10,
		MaxLatency:       10 * time.Second,
		CheckInterval:    30 * time.Second,
		RecoveryDelay:    60 * time.Second,
		Enabled:          true,
	}
}

type EngineHealth struct {
	Name               string    `json:"name"`
	Healthy            bool      `json:"healthy"`
	ConsecutiveFails   int       `json:"consecutive_fails"`
	ConsecutiveSuccess int       `json:"consecutive_success"`
	LastSuccess        time.Time `json:"last_success"`
	LastFailure        time.Time `json:"last_failure"`
	LastError          string    `json:"last_error,omitempty"`
	TotalRequests      int64     `json:"total_requests"`
	TotalFailures      int64     `json:"total_failures"`
	TotalSuccesses     int64     `json:"total_successes"`

	LatencyHistory []time.Duration `json:"-"`
	AvgLatency     time.Duration   `json:"avg_latency"`
	P95Latency     time.Duration   `json:"p95_latency"`

	MarkedUnhealthyAt *time.Time `json:"marked_unhealthy_at,omitempty"`
	MarkedHealthyAt   *time.Time `json:"marked_healthy_at,omitempty"`
}

type HealthMonitor struct {
	mu       sync.RWMutex
	config   HealthConfig
	engines  map[string]*EngineHealth
	stopChan chan struct{}
}

func NewHealthMonitor(config HealthConfig) *HealthMonitor {
	hm := &HealthMonitor{
		config:   config,
		engines:  make(map[string]*EngineHealth),
		stopChan: make(chan struct{}),
	}

	return hm
}

func (hm *HealthMonitor) RegisterEngine(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.engines[name]; !exists {
		now := time.Now()
		hm.engines[name] = &EngineHealth{
			Name:            name,
			Healthy:         true,
			LastSuccess:     now,
			LatencyHistory:  make([]time.Duration, 0, hm.config.LatencyWindow),
			MarkedHealthyAt: &now,
		}
	}
}

func (hm *HealthMonitor) RecordSuccess(name string, latency time.Duration) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	health, exists := hm.engines[name]
	if !exists {
		health = &EngineHealth{
			Name:           name,
			Healthy:        true,
			LatencyHistory: make([]time.Duration, 0, hm.config.LatencyWindow),
		}
		hm.engines[name] = health
	}

	health.TotalRequests++
	health.TotalSuccesses++
	health.LastSuccess = time.Now()
	health.ConsecutiveFails = 0
	health.ConsecutiveSuccess++
	health.LastError = ""

	hm.recordLatency(health, latency)

	if !health.Healthy && health.ConsecutiveSuccess >= hm.config.SuccessThreshold {
		health.Healthy = true
		now := time.Now()
		health.MarkedHealthyAt = &now
		health.MarkedUnhealthyAt = nil
	}
}

func (hm *HealthMonitor) RecordFailure(name string, err error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	health, exists := hm.engines[name]
	if !exists {
		health = &EngineHealth{
			Name:           name,
			Healthy:        true,
			LatencyHistory: make([]time.Duration, 0, hm.config.LatencyWindow),
		}
		hm.engines[name] = health
	}

	health.TotalRequests++
	health.TotalFailures++
	health.LastFailure = time.Now()
	health.ConsecutiveSuccess = 0
	health.ConsecutiveFails++

	if err != nil {
		health.LastError = err.Error()
	}

	if health.Healthy && health.ConsecutiveFails >= hm.config.FailureThreshold {
		health.Healthy = false
		now := time.Now()
		health.MarkedUnhealthyAt = &now
		health.MarkedHealthyAt = nil
	}
}

func (hm *HealthMonitor) recordLatency(health *EngineHealth, latency time.Duration) {
	health.LatencyHistory = append(health.LatencyHistory, latency)

	if len(health.LatencyHistory) > hm.config.LatencyWindow {
		health.LatencyHistory = health.LatencyHistory[1:]
	}

	health.AvgLatency = hm.calculateAvgLatency(health.LatencyHistory)
	health.P95Latency = hm.calculateP95Latency(health.LatencyHistory)
}

func (hm *HealthMonitor) calculateAvgLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, l := range latencies {
		total += l
	}

	return total / time.Duration(len(latencies))
}

func (hm *HealthMonitor) calculateP95Latency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	idx := int(float64(len(sorted)) * 0.95)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}

func (hm *HealthMonitor) IsHealthy(name string) bool {
	if !hm.config.Enabled {
		return true
	}

	hm.mu.RLock()
	defer hm.mu.RUnlock()

	health, exists := hm.engines[name]
	if !exists {
		return true
	}

	return health.Healthy
}

func (hm *HealthMonitor) GetHealth(name string) *EngineHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	health, exists := hm.engines[name]
	if !exists {
		return nil
	}

	healthCopy := *health
	return &healthCopy
}

func (hm *HealthMonitor) GetAllHealth() map[string]*EngineHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result := make(map[string]*EngineHealth)
	for name, health := range hm.engines {
		healthCopy := *health
		result[name] = &healthCopy
	}

	return result
}

func (hm *HealthMonitor) GetHealthyEngines() []string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var healthy []string
	for name, health := range hm.engines {
		if health.Healthy {
			healthy = append(healthy, name)
		}
	}

	return healthy
}

func (hm *HealthMonitor) GetUnhealthyEngines() []string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var unhealthy []string
	for name, health := range hm.engines {
		if !health.Healthy {
			unhealthy = append(unhealthy, name)
		}
	}

	return unhealthy
}

func (hm *HealthMonitor) ResetEngine(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if health, exists := hm.engines[name]; exists {
		now := time.Now()
		health.Healthy = true
		health.ConsecutiveFails = 0
		health.ConsecutiveSuccess = 0
		health.LastError = ""
		health.LatencyHistory = health.LatencyHistory[:0]
		health.MarkedHealthyAt = &now
		health.MarkedUnhealthyAt = nil
	}
}

func (hm *HealthMonitor) ResetAll() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	now := time.Now()
	for _, health := range hm.engines {
		health.Healthy = true
		health.ConsecutiveFails = 0
		health.ConsecutiveSuccess = 0
		health.LastError = ""
		health.LatencyHistory = health.LatencyHistory[:0]
		health.MarkedHealthyAt = &now
		health.MarkedUnhealthyAt = nil
	}
}

func (hm *HealthMonitor) SetConfig(config HealthConfig) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.config = config
}

func (hm *HealthMonitor) Stats() HealthStats {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	stats := HealthStats{
		TotalEngines: len(hm.engines),
		EngineStats:  make(map[string]EngineStats),
	}

	for name, health := range hm.engines {
		if health.Healthy {
			stats.HealthyEngines++
		} else {
			stats.UnhealthyEngines++
		}

		stats.TotalRequests += health.TotalRequests
		stats.TotalSuccesses += health.TotalSuccesses
		stats.TotalFailures += health.TotalFailures

		stats.EngineStats[name] = EngineStats{
			Healthy:       health.Healthy,
			SuccessRate:   float64(health.TotalSuccesses) / max(float64(health.TotalRequests), 1),
			AvgLatency:    health.AvgLatency,
			TotalRequests: health.TotalRequests,
		}
	}

	if stats.TotalRequests > 0 {
		stats.OverallSuccessRate = float64(stats.TotalSuccesses) / float64(stats.TotalRequests)
	}

	return stats
}

type HealthStats struct {
	TotalEngines       int                    `json:"total_engines"`
	HealthyEngines     int                    `json:"healthy_engines"`
	UnhealthyEngines   int                    `json:"unhealthy_engines"`
	TotalRequests      int64                  `json:"total_requests"`
	TotalSuccesses     int64                  `json:"total_successes"`
	TotalFailures      int64                  `json:"total_failures"`
	OverallSuccessRate float64                `json:"overall_success_rate"`
	EngineStats        map[string]EngineStats `json:"engine_stats"`
}

type EngineStats struct {
	Healthy       bool          `json:"healthy"`
	SuccessRate   float64       `json:"success_rate"`
	AvgLatency    time.Duration `json:"avg_latency"`
	TotalRequests int64         `json:"total_requests"`
}

func (hm *HealthMonitor) ShouldRetry(name string) bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	health, exists := hm.engines[name]
	if !exists {
		return true
	}

	if health.ConsecutiveFails >= hm.config.FailureThreshold {
		if health.MarkedUnhealthyAt != nil {
			if time.Since(*health.MarkedUnhealthyAt) < hm.config.RecoveryDelay {
				return false
			}
		}
	}

	return true
}

func (hm *HealthMonitor) CheckLatencyThreshold(name string) bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	health, exists := hm.engines[name]
	if !exists {
		return true
	}

	return health.AvgLatency <= hm.config.MaxLatency
}
