package privacy

import (
	"math/rand"
	"net/http"
	"time"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
}

func RandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

func TimingJitter(min, max time.Duration) {
	if max <= min {
		time.Sleep(min)
		return
	}
	jitter := min + time.Duration(rand.Int63n(int64(max-min)))
	time.Sleep(jitter)
}

func AnonymizeRequest(req *http.Request) {
	req.Header.Set("User-Agent", RandomUserAgent())

	req.Header.Del("X-Forwarded-For")
	req.Header.Del("X-Real-IP")
	req.Header.Del("X-Forwarded-Host")
	req.Header.Del("X-Forwarded-Proto")

	req.Header.Del("Sec-CH-UA")
	req.Header.Del("Sec-CH-UA-Mobile")
	req.Header.Del("Sec-CH-UA-Platform")
	req.Header.Del("Sec-CH-UA-Full-Version")
	req.Header.Del("Sec-CH-UA-Arch")
	req.Header.Del("Sec-CH-UA-Model")

	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("DNT", "1")
	req.Header.Set("Sec-GPC", "1")
}

var headersToRemove = []string{
	"X-Forwarded-For",
	"X-Real-IP",
	"X-Forwarded-Host",
	"X-Forwarded-Proto",
	"Sec-CH-UA",
	"Sec-CH-UA-Mobile",
	"Sec-CH-UA-Platform",
	"Sec-CH-UA-Full-Version",
	"Sec-CH-UA-Arch",
	"Sec-CH-UA-Model",
}
