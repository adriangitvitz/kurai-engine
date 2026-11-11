package blocklist

import (
	"bufio"
	"io"
	"strings"
)

func ParseHosts(r io.Reader, store *DomainStore) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			continue
		}

		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		ip := fields[0]

		if !isBlockingIP(ip) {
			continue
		}

		for _, domain := range fields[1:] {
			if isLocalhost(domain) {
				continue
			}

			store.Add(domain)
		}
	}

	return scanner.Err()
}

func isBlockingIP(ip string) bool {
	switch ip {
	case "0.0.0.0", "127.0.0.1", "::", "::1":
		return true
	}
	return false
}

func isLocalhost(domain string) bool {
	domain = strings.ToLower(domain)
	switch domain {
	case "localhost", "localhost.localdomain", "local", "broadcasthost",
		"0.0.0.0", "127.0.0.1", "::1", "ip6-localhost", "ip6-loopback":
		return true
	}
	return false
}
