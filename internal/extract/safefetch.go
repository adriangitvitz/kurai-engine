package extract

import (
	"net"
	"net/url"
	"strings"
)

func IsInternalURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "" {
		return false
	}

	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()
	}

	if !strings.Contains(host, ".") {
		return true
	}

	for _, suffix := range []string{".local", ".internal", ".docker", ".localhost"} {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}
