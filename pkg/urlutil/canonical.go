package urlutil

import (
	"net/url"
	"path"
	"sort"
	"strings"
)

func Canonicalize(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return strings.ToLower(rawURL)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme == "http" {
		scheme = "https"
	}

	host := strings.ToLower(u.Hostname())
	for _, prefix := range []string{"www.", "m.", "amp."} {
		host = strings.TrimPrefix(host, prefix)
	}

	cleaned := path.Clean(u.Path)
	if cleaned == "." {
		cleaned = "/"
	}
	if cleaned != "/" {
		cleaned = strings.TrimRight(cleaned, "/")
	}

	q := u.Query()
	for k := range q {
		if IsTrackingParam(k) {
			delete(q, k)
		}
	}
	rawQuery := buildSortedQuery(q)

	out := scheme + "://" + host + cleaned
	if rawQuery != "" {
		out += "?" + rawQuery
	}
	return out
}

func CanonicalKey(rawURL string) string {
	c := Canonicalize(rawURL)
	indexSuffixes := []string{"/index.html", "/index.htm", "/index.php", "/default.aspx", "/default.htm"}
	for _, suffix := range indexSuffixes {
		if strings.HasSuffix(c, suffix) {
			trimmed := strings.TrimSuffix(c, suffix)
			schemeEnd := strings.Index(trimmed, "://")
			if schemeEnd >= 0 && !strings.ContainsRune(trimmed[schemeEnd+3:], '/') {
				c = trimmed + "/"
			} else {
				c = trimmed
			}
			break
		}
	}
	return c
}

func buildSortedQuery(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		for _, v := range q[k] {
			if b.Len() > 0 {
				b.WriteByte('&')
			}
			b.WriteString(url.QueryEscape(k))
			b.WriteByte('=')
			b.WriteString(url.QueryEscape(v))
		}
	}
	return b.String()
}
