package extract

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

const sampleArticleHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>What Is Ownership?</title>
  <meta name="author" content="Steve Klabnik">
  <meta property="article:published_time" content="2024-09-15T12:00:00Z">
  <meta property="og:site_name" content="The Rust Programming Language">
</head>
<body>
  <header><nav>Skip nav</nav></header>
  <main>
    <article>
      <h1>What Is Ownership?</h1>
      <p>Ownership is a set of rules that govern how a Rust program manages memory.</p>
      <p>All programs have to manage the way they use a computer's memory while running.</p>
      <h2>Ownership Rules</h2>
      <ul>
        <li>Each value in Rust has an <em>owner</em>.</li>
        <li>There can only be one owner at a time.</li>
        <li>When the owner goes out of scope, the value will be dropped.</li>
      </ul>
      <p>For more, see <a href="https://doc.rust-lang.org/book/ch04-02-references-and-borrowing.html">References and Borrowing</a>.</p>
    </article>
  </main>
  <footer>© 2024 The Rust Foundation</footer>
</body>
</html>`

func newTestFetcher(t *testing.T) *Fetcher {
	t.Helper()
	return NewFetcher(Config{
		Timeout:              5 * time.Second,
		MaxBytes:             1 << 20,
		MaxRedirects:         5,
		UserAgent:            "kurai-test/1.0",
		AllowPrivateNetworks: true,
	})
}

func TestFetch_ExtractsTitleBylineDateAndMarkdown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(sampleArticleHTML))
	}))
	defer srv.Close()

	f := newTestFetcher(t)
	got, err := f.Fetch(context.Background(), srv.URL+"/article", FetchOpts{Format: FormatMarkdown})
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, "What Is Ownership?", got.Title)
	assert.Equal(t, "Steve Klabnik", got.Byline)
	require.NotNil(t, got.PublishedDate, "published date must parse from article:published_time")
	assert.Equal(t, 2024, got.PublishedDate.Year())
	assert.Equal(t, "The Rust Programming Language", got.SiteName)
	assert.Contains(t, strings.ToLower(got.Language), "en")
	assert.NotEmpty(t, got.ContentMarkdown, "markdown body must be populated")
	assert.Contains(t, got.ContentMarkdown, "Ownership is a set of rules")
	assert.Contains(t, got.ContentMarkdown, "Ownership Rules")
	assert.NotContains(t, got.ContentMarkdown, "Skip nav", "boilerplate header must be stripped")
	assert.Greater(t, got.WordCount, 30)
	assert.NotEmpty(t, got.Excerpt)
	assert.Equal(t, srv.URL+"/article", got.URL)
	assert.Equal(t, srv.URL+"/article", got.FinalURL)
}

func TestFetch_FormatTextOmitsMarkdown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(sampleArticleHTML))
	}))
	defer srv.Close()

	f := newTestFetcher(t)
	got, err := f.Fetch(context.Background(), srv.URL, FetchOpts{Format: FormatText})
	require.NoError(t, err)
	assert.Empty(t, got.ContentMarkdown, "Format=text must not include markdown")
	assert.NotEmpty(t, got.ContentText)
}

func TestFetch_FormatBothEmitsBoth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(sampleArticleHTML))
	}))
	defer srv.Close()

	f := newTestFetcher(t)
	got, err := f.Fetch(context.Background(), srv.URL, FetchOpts{Format: FormatBoth})
	require.NoError(t, err)
	assert.NotEmpty(t, got.ContentMarkdown)
	assert.NotEmpty(t, got.ContentText)
}

func TestFetch_RejectsPrivateNetworkWhenDisallowed(t *testing.T) {
	f := NewFetcher(Config{
		Timeout:              5 * time.Second,
		MaxBytes:             1 << 20,
		MaxRedirects:         5,
		AllowPrivateNetworks: false,
	})

	_, err := f.Fetch(context.Background(), "http://127.0.0.1:65535/x", FetchOpts{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPrivateAddress), "got %v", err)
}

func TestFetch_RejectsInvalidURL(t *testing.T) {
	f := newTestFetcher(t)
	_, err := f.Fetch(context.Background(), "not-a-url", FetchOpts{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidURL), "got %v", err)
}

func TestFetch_RejectsUnsupportedContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write([]byte("PK\x03\x04..."))
	}))
	defer srv.Close()

	f := newTestFetcher(t)
	_, err := f.Fetch(context.Background(), srv.URL, FetchOpts{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedType), "got %v", err)
}

func servePDF(t *testing.T, body []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func requirePDFTool(t *testing.T) {
	t.Helper()
	if _, err := pdfTool(); err != nil {
		t.Skip("pdftotext not installed")
	}
}

func TestFetch_ExtractsPDF(t *testing.T) {
	requirePDFTool(t)
	body, err := os.ReadFile(filepath.Join("testdata", "minimal.pdf"))
	require.NoError(t, err)

	srv := servePDF(t, body)
	f := newTestFetcher(t)

	art, err := f.Fetch(context.Background(), srv.URL, FetchOpts{})
	require.NoError(t, err, "a PDF must no longer be a dead end")
	assert.Contains(t, art.ContentMarkdown, "Hello kurai extraction")
	assert.Positive(t, art.WordCount)
	assert.NotEmpty(t, art.Excerpt)
}

func TestFetch_PDFFormatText(t *testing.T) {
	requirePDFTool(t)
	body, err := os.ReadFile(filepath.Join("testdata", "minimal.pdf"))
	require.NoError(t, err)

	f := newTestFetcher(t)
	art, err := f.Fetch(context.Background(), servePDF(t, body).URL, FetchOpts{Format: FormatText})
	require.NoError(t, err)
	assert.Contains(t, art.ContentText, "Hello kurai extraction")
}

func TestFetch_MalformedPDFIsExtractionFailure(t *testing.T) {
	requirePDFTool(t)
	srv := servePDF(t, []byte("%PDF-1.7..."))
	f := newTestFetcher(t)

	_, err := f.Fetch(context.Background(), srv.URL, FetchOpts{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrExtractionFailed), "got %v", err)
	assert.False(t, errors.Is(err, ErrUnsupportedType))
}

func TestIsPDFContentType(t *testing.T) {
	for _, ct := range []string{"application/pdf", "Application/PDF; charset=binary", "application/x-pdf"} {
		assert.True(t, isPDFContentType(ct), ct)
	}
	for _, ct := range []string{"text/html", "application/zip", ""} {
		assert.False(t, isPDFContentType(ct), ct)
	}
}

func TestFetch_RejectsResponseTooLarge(t *testing.T) {
	big := strings.Repeat("a", 1<<20)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>" + big + "</body></html>"))
	}))
	defer srv.Close()

	f := NewFetcher(Config{
		Timeout:              5 * time.Second,
		MaxBytes:             4096,
		MaxRedirects:         5,
		AllowPrivateNetworks: true,
	})
	_, err := f.Fetch(context.Background(), srv.URL, FetchOpts{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrResponseTooLarge), "got %v", err)
}

func TestFetch_HonorsRedirectCapAndTracksFinalURL(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/middle", http.StatusFound)
	})
	mux.HandleFunc("/middle", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/final", http.StatusFound)
	})
	mux.HandleFunc("/final", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(sampleArticleHTML))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := newTestFetcher(t)
	got, err := f.Fetch(context.Background(), srv.URL+"/start", FetchOpts{Format: FormatMarkdown})
	require.NoError(t, err)
	assert.Equal(t, srv.URL+"/start", got.URL)
	assert.Equal(t, srv.URL+"/final", got.FinalURL)
}

func TestFetch_FailsWhenRedirectCapExceeded(t *testing.T) {
	mux := http.NewServeMux()
	for i := 0; i < 10; i++ {
		i := i
		mux.HandleFunc(fmt.Sprintf("/r%d", i), func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, fmt.Sprintf("/r%d", i+1), http.StatusFound)
		})
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := NewFetcher(Config{
		Timeout:              5 * time.Second,
		MaxBytes:             1 << 20,
		MaxRedirects:         3,
		AllowPrivateNetworks: true,
	})
	_, err := f.Fetch(context.Background(), srv.URL+"/r0", FetchOpts{})
	require.Error(t, err)
}

func TestFetch_HonorsContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	f := newTestFetcher(t)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := f.Fetch(ctx, srv.URL, FetchOpts{})
	require.Error(t, err, "context timeout must surface as an error")
}

func TestIsInternalURL(t *testing.T) {
	cases := map[string]bool{
		"http://localhost/x":         true,
		"http://127.0.0.1/x":         true,
		"http://[::1]/x":             true,
		"http://10.0.0.5/x":          true,
		"http://192.168.1.1/x":       true,
		"http://172.16.0.1/x":        true,
		"http://kurai/health":        true,
		"http://service.local/x":     true,
		"http://example.com/path":    false,
		"https://doc.rust-lang.org/": false,
	}
	for raw, want := range cases {
		assert.Equal(t, want, IsInternalURL(raw), "url=%s", raw)
	}
}

func TestFetch_SendsBrowserUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(sampleArticleHTML))
	}))
	defer srv.Close()

	f := newTestFetcher(t)
	_, err := f.Fetch(context.Background(), srv.URL, FetchOpts{})
	require.NoError(t, err)
	assert.Contains(t, gotUA, "Mozilla/", "must send a browser-like UA, got %q", gotUA)
	assert.NotContains(t, gotUA, "kurai", "the kurai bot UA gets 403'd by CDNs/StackOverflow")
}

func TestFetch_ReadsContentServedInsideNoscript(t *testing.T) {
	post := strings.Repeat("<p>The foundation does not plan to change signature schemes before the standard settles, and validators keep their current keys.</p>\n", 8)
	page := `<!doctype html><html><head><title>Position on post-quantum cryptography</title></head><body>
<section id="splash"><style>html { overflow-y: hidden !important; }</style></section>
<noscript><iframe src="https://tags.example/ns.html" height="0" width="0"></iframe></noscript>
<div id="app"></div>
<noscript data-path="/t/position/1">
<header><a href="/">Forum</a></header>
<div id="main-outlet" role="main"><h1>Position on post-quantum cryptography</h1>
<div class="post" itemprop="text">` + post + `</div></div>
</noscript>
<script>window.boot()</script></body></html>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()

	article, err := newTestFetcher(t).Fetch(context.Background(), srv.URL, FetchOpts{Format: FormatBoth})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	for _, body := range []string{article.ContentMarkdown, article.ContentText} {
		if !strings.Contains(body, "validators keep their current keys") {
			t.Fatalf("noscript content missing from %q", body)
		}
		for _, markup := range []string{"<div", "&lt;", "<p>", "itemprop", "tags.example", "overflow-y"} {
			if strings.Contains(body, markup) {
				t.Fatalf("raw markup %q leaked into %q", markup, body)
			}
		}
	}
}

func TestPromoteNoscriptContent_LeavesSmallBlocksAlone(t *testing.T) {
	doc, err := html.ParseWithOptions(strings.NewReader(`<html><body><p>Visible article text.</p><noscript><img src="/pixel.gif"></noscript></body></html>`), html.ParseOptionEnableScripting(false))
	if err != nil {
		t.Fatal(err)
	}
	promoteNoscriptContent(doc)
	var buf strings.Builder
	_ = html.Render(&buf, doc)
	if !strings.Contains(buf.String(), "<noscript><img") {
		t.Fatalf("a small noscript block must stay wrapped: %s", buf.String())
	}
}

func TestFetch_QAPageAnswersFollowTheQuestion(t *testing.T) {
	question := strings.Repeat("<p>We are asked about our position on post-quantum support and the threat to the network.</p>\n", 6)
	page := `<!doctype html><html><head><title>Position on PQC</title></head><body>
<div itemscope itemtype="https://schema.org/QAPage"><div itemprop="mainEntity" itemscope itemtype="https://schema.org/Question">
<h1 itemprop="name">Position on PQC</h1>
<div id="post_1"><div class="post" itemprop="text">` + question + `</div></div>
<div id="post_2" itemprop="suggestedAnswer" itemscope itemtype="https://schema.org/Answer">
<span itemprop="author" itemscope itemtype="http://schema.org/Person"><span itemprop="name">nycnewman</span></span>
<time itemprop="datePublished" datetime="2026-08-14T15:46:42Z">August 14, 2026</time>
<div class="post" itemprop="text"><p>The national standards body published its migration timeline this month.</p></div>
</div>
<div id="post_3" itemprop="acceptedAnswer" itemscope itemtype="https://schema.org/Answer">
<div class="post" itemprop="text"><p>Validators will rotate to hybrid keys once the library ships.</p></div>
</div>
</div></div></body></html>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()

	article, err := newTestFetcher(t).Fetch(context.Background(), srv.URL, FetchOpts{Format: FormatBoth})
	require.NoError(t, err)
	for _, body := range []string{article.ContentMarkdown, article.ContentText} {
		assert.Contains(t, body, "threat to the network")
		assert.Contains(t, body, "nycnewman · 2026-08-14")
		assert.Equal(t, 1, strings.Count(body, "published its migration timeline"))
		assert.Equal(t, 1, strings.Count(body, "rotate to hybrid keys"))
		assert.Less(t, strings.Index(body, "threat to the network"), strings.Index(body, "migration timeline"))
	}
	assert.Greater(t, article.WordCount, 100)
}
