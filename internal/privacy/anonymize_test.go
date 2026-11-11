package privacy

import (
	"net/http"
	"testing"
	"time"
)

func TestRandomUserAgent(t *testing.T) {
	ua := RandomUserAgent()
	if ua == "" {
		t.Error("RandomUserAgent returned empty string")
	}

	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		seen[RandomUserAgent()] = true
	}
	if len(seen) < 2 {
		t.Errorf("RandomUserAgent should return varied results, got %d unique values", len(seen))
	}
}

func TestRandomUserAgentValid(t *testing.T) {
	for i := 0; i < 20; i++ {
		ua := RandomUserAgent()
		if len(ua) < 50 {
			t.Errorf("User agent too short to be realistic: %s", ua)
		}
		if !containsString(ua, "Mozilla") {
			t.Errorf("User agent should contain Mozilla: %s", ua)
		}
	}
}

func TestTimingJitter(t *testing.T) {
	min := 10 * time.Millisecond
	max := 50 * time.Millisecond

	start := time.Now()
	TimingJitter(min, max)
	elapsed := time.Since(start)

	if elapsed < min {
		t.Errorf("TimingJitter slept less than min: %v < %v", elapsed, min)
	}
	if elapsed > max+10*time.Millisecond {
		t.Errorf("TimingJitter slept more than max: %v > %v", elapsed, max)
	}
}

func TestTimingJitterVariation(t *testing.T) {
	min := 5 * time.Millisecond
	max := 20 * time.Millisecond

	var durations []time.Duration
	for i := 0; i < 10; i++ {
		start := time.Now()
		TimingJitter(min, max)
		durations = append(durations, time.Since(start))
	}

	allSame := true
	first := durations[0].Truncate(time.Millisecond)
	for _, d := range durations[1:] {
		if d.Truncate(time.Millisecond) != first {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("TimingJitter should produce varied delays")
	}
}

func TestAnonymizeRequest(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Real-IP", "5.6.7.8")
	req.Header.Set("Sec-CH-UA", "Chrome")
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Platform", "Windows")

	AnonymizeRequest(req)

	if req.Header.Get("X-Forwarded-For") != "" {
		t.Error("X-Forwarded-For should be removed")
	}
	if req.Header.Get("X-Real-IP") != "" {
		t.Error("X-Real-IP should be removed")
	}
	if req.Header.Get("Sec-CH-UA") != "" {
		t.Error("Sec-CH-UA should be removed")
	}
	if req.Header.Get("Sec-CH-UA-Mobile") != "" {
		t.Error("Sec-CH-UA-Mobile should be removed")
	}
	if req.Header.Get("Sec-CH-UA-Platform") != "" {
		t.Error("Sec-CH-UA-Platform should be removed")
	}

	ua := req.Header.Get("User-Agent")
	if ua == "" {
		t.Error("User-Agent should be set")
	}
	if !containsString(ua, "Mozilla") {
		t.Error("User-Agent should be a realistic browser UA")
	}

	if req.Header.Get("Accept-Language") != "en-US,en;q=0.9" {
		t.Error("Accept-Language should be set")
	}
	if req.Header.Get("DNT") != "1" {
		t.Error("DNT should be set to 1")
	}
	if req.Header.Get("Sec-GPC") != "1" {
		t.Error("Sec-GPC should be set to 1")
	}
}

func TestAnonymizeRequestPreservesMethod(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://example.com", nil)
	req.Header.Set("Content-Type", "application/json")

	AnonymizeRequest(req)

	if req.Header.Get("Content-Type") != "application/json" {
		t.Error("Content-Type should be preserved")
	}
	if req.Method != "POST" {
		t.Error("Method should be preserved")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
