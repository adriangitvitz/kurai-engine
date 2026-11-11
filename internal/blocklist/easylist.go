package blocklist

import (
	"bufio"
	"io"
	"strings"
)

func ParseEasyList(r io.Reader, store *DomainStore) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "!") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			continue
		}

		if strings.HasPrefix(line, "@@") {
			continue
		}

		if strings.Contains(line, "##") || strings.Contains(line, "#@#") || strings.Contains(line, "#?#") {
			continue
		}

		if strings.HasPrefix(line, "/") && strings.HasSuffix(line, "/") {
			continue
		}

		if strings.HasPrefix(line, "||") && strings.HasSuffix(line, "^") {
			domain := strings.TrimPrefix(line, "||")
			domain = strings.TrimSuffix(domain, "^")

			if strings.Contains(domain, "/") {
				continue
			}

			if strings.Contains(domain, "*") {
				continue
			}

			if strings.Contains(domain, "$") {
				continue
			}

			if strings.Contains(domain, "|") {
				continue
			}

			if domain != "" {
				store.Add(domain)
			}
		}
	}

	return scanner.Err()
}
