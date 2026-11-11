package privacy

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestNewDoHResolver(t *testing.T) {
	tests := []struct {
		name        string
		resolverURL string
		wantErr     bool
	}{
		{
			name:        "cloudflare resolver (IP)",
			resolverURL: "https://1.1.1.1/dns-query",
			wantErr:     false,
		},
		{
			name:        "google resolver (IP)",
			resolverURL: "https://8.8.8.8/dns-query",
			wantErr:     false,
		},
		{
			name:        "quad9 resolver (IP)",
			resolverURL: "https://9.9.9.9:5053/dns-query",
			wantErr:     false,
		},
		{
			name:        "empty resolver URL uses default",
			resolverURL: "",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewDoHResolver(tt.resolverURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDoHResolver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && resolver == nil {
				t.Error("NewDoHResolver() returned nil resolver")
			}
		})
	}
}

func TestDoHResolver_LookupHost(t *testing.T) {
	resolver, err := NewDoHResolver("")
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	addrs, err := resolver.LookupHost(ctx, "example.com")
	if err != nil {
		t.Fatalf("LookupHost() error = %v", err)
	}

	if len(addrs) == 0 {
		t.Error("LookupHost() returned no addresses")
	}

	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			t.Errorf("LookupHost() returned invalid IP: %s", addr)
		}
	}
}

func TestDoHResolver_LookupHost_InvalidDomain(t *testing.T) {
	resolver, err := NewDoHResolver("")
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = resolver.LookupHost(ctx, "this-domain-definitely-does-not-exist-xyz123.com")
	if err == nil {
		t.Error("LookupHost() expected error for non-existent domain")
	}
}

func TestNewDoHClient(t *testing.T) {
	client, err := NewDoHClient("")
	if err != nil {
		t.Fatalf("NewDoHClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewDoHClient() returned nil client")
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Client transport is not *http.Transport")
	}

	if transport.DialContext == nil {
		t.Error("Client transport DialContext is nil")
	}
}

func TestDoHClient_Request(t *testing.T) {
	client, err := NewDoHClient("")
	if err != nil {
		t.Fatalf("NewDoHClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("DoHClient request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestDoHClient_WithCustomResolver(t *testing.T) {
	client, err := NewDoHClient("https://8.8.8.8/dns-query")
	if err != nil {
		t.Fatalf("NewDoHClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("DoHClient request with Google DoH failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestDefaultDoHResolverURL(t *testing.T) {
	url := DefaultDoHResolverURL()
	if url == "" {
		t.Error("DefaultDoHResolverURL() returned empty string")
	}
	if url != "https://1.1.1.1/dns-query" {
		t.Errorf("DefaultDoHResolverURL() = %s, want https://1.1.1.1/dns-query", url)
	}
}
