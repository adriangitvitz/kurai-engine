package privacy

import (
	"net"
	"net/http"
	"time"

	"github.com/ncruces/go-dns"
)

const defaultDoHResolver = "https://1.1.1.1/dns-query"

var DoHResolvers = map[string]string{
	"cloudflare":  "https://1.1.1.1/dns-query",
	"cloudflare2": "https://1.0.0.1/dns-query",
	"quad9":       "https://9.9.9.9:5053/dns-query",
	"google":      "https://8.8.8.8/dns-query",
}

func DefaultDoHResolverURL() string {
	return defaultDoHResolver
}

func NewDoHResolver(resolverURL string) (*net.Resolver, error) {
	if resolverURL == "" {
		resolverURL = defaultDoHResolver
	}

	return dns.NewDoHResolver(resolverURL)
}

func NewDoHClient(resolverURL string) (*http.Client, error) {
	resolver, err := NewDoHResolver(resolverURL)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{
		Resolver: resolver,
		Timeout:  30 * time.Second,
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return client, nil
}
