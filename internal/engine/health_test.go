package engine

import (
	"errors"
	"testing"
	"time"
)

func TestHealthMonitor_RegisterEngine(t *testing.T) {
	hm := NewHealthMonitor(DefaultHealthConfig())

	hm.RegisterEngine("google")
	hm.RegisterEngine("bing")

	health := hm.GetAllHealth()
	if len(health) != 2 {
		t.Errorf("GetAllHealth() got %d engines, want 2", len(health))
	}

	if !hm.IsHealthy("google") {
		t.Error("New engine should be healthy by default")
	}
}

func TestHealthMonitor_RecordSuccess(t *testing.T) {
	hm := NewHealthMonitor(DefaultHealthConfig())
	hm.RegisterEngine("test")

	hm.RecordSuccess("test", 100*time.Millisecond)
	hm.RecordSuccess("test", 150*time.Millisecond)
	hm.RecordSuccess("test", 200*time.Millisecond)

	health := hm.GetHealth("test")
	if health == nil {
		t.Fatal("GetHealth() returned nil")
	}

	if health.TotalSuccesses != 3 {
		t.Errorf("TotalSuccesses = %d, want 3", health.TotalSuccesses)
	}
	if health.TotalRequests != 3 {
		t.Errorf("TotalRequests = %d, want 3", health.TotalRequests)
	}
	if health.ConsecutiveSuccess != 3 {
		t.Errorf("ConsecutiveSuccess = %d, want 3", health.ConsecutiveSuccess)
	}
}

func TestHealthMonitor_RecordFailure(t *testing.T) {
	config := DefaultHealthConfig()
	config.FailureThreshold = 3
	hm := NewHealthMonitor(config)
	hm.RegisterEngine("test")

	hm.RecordFailure("test", errors.New("error 1"))
	hm.RecordFailure("test", errors.New("error 2"))

	if !hm.IsHealthy("test") {
		t.Error("Engine should still be healthy after 2 failures")
	}

	hm.RecordFailure("test", errors.New("error 3"))

	if hm.IsHealthy("test") {
		t.Error("Engine should be unhealthy after 3 consecutive failures")
	}

	health := hm.GetHealth("test")
	if health.LastError != "error 3" {
		t.Errorf("LastError = %q, want %q", health.LastError, "error 3")
	}
}

func TestHealthMonitor_RecoveryAfterSuccess(t *testing.T) {
	config := DefaultHealthConfig()
	config.FailureThreshold = 2
	config.SuccessThreshold = 2
	hm := NewHealthMonitor(config)
	hm.RegisterEngine("test")

	hm.RecordFailure("test", nil)
	hm.RecordFailure("test", nil)

	if hm.IsHealthy("test") {
		t.Error("Engine should be unhealthy")
	}

	hm.RecordSuccess("test", 100*time.Millisecond)
	if hm.IsHealthy("test") {
		t.Error("Engine should still be unhealthy after 1 success")
	}

	hm.RecordSuccess("test", 100*time.Millisecond)
	if !hm.IsHealthy("test") {
		t.Error("Engine should be healthy after 2 consecutive successes")
	}
}

func TestHealthMonitor_LatencyTracking(t *testing.T) {
	config := DefaultHealthConfig()
	config.LatencyWindow = 5
	hm := NewHealthMonitor(config)
	hm.RegisterEngine("test")

	latencies := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
		400 * time.Millisecond,
		500 * time.Millisecond,
	}

	for _, l := range latencies {
		hm.RecordSuccess("test", l)
	}

	health := hm.GetHealth("test")

	expectedAvg := 300 * time.Millisecond
	if health.AvgLatency != expectedAvg {
		t.Errorf("AvgLatency = %v, want %v", health.AvgLatency, expectedAvg)
	}

	if health.P95Latency < 400*time.Millisecond {
		t.Errorf("P95Latency = %v, want >= 400ms", health.P95Latency)
	}
}

func TestHealthMonitor_GetHealthyEngines(t *testing.T) {
	config := DefaultHealthConfig()
	config.FailureThreshold = 2
	hm := NewHealthMonitor(config)

	hm.RegisterEngine("healthy1")
	hm.RegisterEngine("healthy2")
	hm.RegisterEngine("unhealthy")

	hm.RecordFailure("unhealthy", nil)
	hm.RecordFailure("unhealthy", nil)

	healthy := hm.GetHealthyEngines()
	if len(healthy) != 2 {
		t.Errorf("GetHealthyEngines() got %d, want 2", len(healthy))
	}

	unhealthy := hm.GetUnhealthyEngines()
	if len(unhealthy) != 1 {
		t.Errorf("GetUnhealthyEngines() got %d, want 1", len(unhealthy))
	}
}

func TestHealthMonitor_ResetEngine(t *testing.T) {
	config := DefaultHealthConfig()
	config.FailureThreshold = 2
	hm := NewHealthMonitor(config)
	hm.RegisterEngine("test")

	hm.RecordFailure("test", nil)
	hm.RecordFailure("test", nil)

	if hm.IsHealthy("test") {
		t.Error("Engine should be unhealthy")
	}

	hm.ResetEngine("test")

	if !hm.IsHealthy("test") {
		t.Error("Engine should be healthy after reset")
	}
}

func TestHealthMonitor_ResetAll(t *testing.T) {
	config := DefaultHealthConfig()
	config.FailureThreshold = 1
	hm := NewHealthMonitor(config)

	hm.RegisterEngine("test1")
	hm.RegisterEngine("test2")

	hm.RecordFailure("test1", nil)
	hm.RecordFailure("test2", nil)

	unhealthy := hm.GetUnhealthyEngines()
	if len(unhealthy) != 2 {
		t.Errorf("Expected 2 unhealthy engines, got %d", len(unhealthy))
	}

	hm.ResetAll()

	healthy := hm.GetHealthyEngines()
	if len(healthy) != 2 {
		t.Errorf("Expected 2 healthy engines after reset, got %d", len(healthy))
	}
}

func TestHealthMonitor_Stats(t *testing.T) {
	hm := NewHealthMonitor(DefaultHealthConfig())

	hm.RegisterEngine("engine1")
	hm.RegisterEngine("engine2")

	hm.RecordSuccess("engine1", 100*time.Millisecond)
	hm.RecordSuccess("engine1", 100*time.Millisecond)
	hm.RecordFailure("engine2", nil)

	stats := hm.Stats()

	if stats.TotalEngines != 2 {
		t.Errorf("TotalEngines = %d, want 2", stats.TotalEngines)
	}
	if stats.TotalRequests != 3 {
		t.Errorf("TotalRequests = %d, want 3", stats.TotalRequests)
	}
	if stats.TotalSuccesses != 2 {
		t.Errorf("TotalSuccesses = %d, want 2", stats.TotalSuccesses)
	}
	if stats.TotalFailures != 1 {
		t.Errorf("TotalFailures = %d, want 1", stats.TotalFailures)
	}
}

func TestHealthMonitor_UnknownEngineHealthy(t *testing.T) {
	hm := NewHealthMonitor(DefaultHealthConfig())

	if !hm.IsHealthy("unknown") {
		t.Error("Unknown engine should be considered healthy")
	}
}

func TestHealthMonitor_DisabledAlwaysHealthy(t *testing.T) {
	config := DefaultHealthConfig()
	config.Enabled = false
	hm := NewHealthMonitor(config)

	hm.RegisterEngine("test")

	for i := 0; i < 10; i++ {
		hm.RecordFailure("test", nil)
	}

	if !hm.IsHealthy("test") {
		t.Error("Engine should be healthy when monitoring is disabled")
	}
}

func TestHealthMonitor_ShouldRetry(t *testing.T) {
	config := DefaultHealthConfig()
	config.FailureThreshold = 2
	config.RecoveryDelay = 1 * time.Second
	hm := NewHealthMonitor(config)
	hm.RegisterEngine("test")

	if !hm.ShouldRetry("test") {
		t.Error("Should retry initially")
	}

	hm.RecordFailure("test", nil)
	hm.RecordFailure("test", nil)

	if hm.ShouldRetry("test") {
		t.Error("Should not retry immediately after marking unhealthy")
	}
}

func TestHealthMonitor_CheckLatencyThreshold(t *testing.T) {
	config := DefaultHealthConfig()
	config.MaxLatency = 500 * time.Millisecond
	hm := NewHealthMonitor(config)
	hm.RegisterEngine("fast")
	hm.RegisterEngine("slow")

	hm.RecordSuccess("fast", 100*time.Millisecond)
	hm.RecordSuccess("slow", 1000*time.Millisecond)

	if !hm.CheckLatencyThreshold("fast") {
		t.Error("Fast engine should pass latency threshold")
	}

	if hm.CheckLatencyThreshold("slow") {
		t.Error("Slow engine should fail latency threshold")
	}
}
