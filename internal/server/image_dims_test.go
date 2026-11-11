package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/searxng"
)

func TestConvertToImageResults_ParsesResolutionField(t *testing.T) {
	results := []searxng.Result{
		{
			URL:        "https://example.com/page",
			Title:      "K8s diagram",
			ImgSrc:     "https://example.com/img.svg",
			Resolution: "1352 x 649",
			ImgFormat:  "SVG",
			Engines:    []string{"google images"},
		},
	}
	imgs := convertToImageResults(results)
	require.Len(t, imgs, 1)
	assert.Equal(t, 1352, imgs[0].Width, "Width must come from SearXNG resolution field")
	assert.Equal(t, 649, imgs[0].Height, "Height must come from SearXNG resolution field")
}

func TestConvertToImageResults_AlternateResolutionFormats(t *testing.T) {
	cases := []struct {
		raw  string
		w, h int
	}{
		{"1920x1080", 1920, 1080},
		{"1920 x 1080", 1920, 1080},
		{"1920×1080", 1920, 1080},
		{"800X600", 800, 600},
		{" 4096  x  2160 ", 4096, 2160},
	}
	for _, c := range cases {
		results := []searxng.Result{
			{URL: "https://x/", ImgSrc: "https://x/img.png", Resolution: c.raw},
		}
		got := convertToImageResults(results)
		require.Len(t, got, 1, "raw=%q", c.raw)
		assert.Equal(t, c.w, got[0].Width, "Width for resolution=%q", c.raw)
		assert.Equal(t, c.h, got[0].Height, "Height for resolution=%q", c.raw)
	}
}

func TestConvertToImageResults_FallsBackToContentHeuristic(t *testing.T) {
	results := []searxng.Result{
		{URL: "https://x/", ImgSrc: "https://x/img.jpg", Content: "Photo of a cat 800x600 high quality"},
	}
	got := convertToImageResults(results)
	require.Len(t, got, 1)
	assert.Equal(t, 800, got[0].Width)
	assert.Equal(t, 600, got[0].Height)
}

func TestConvertToImageResults_EmptyDimensionsIfNoSignal(t *testing.T) {
	results := []searxng.Result{
		{URL: "https://x/", ImgSrc: "https://x/img.png", Content: "no dimensions in this snippet"},
	}
	got := convertToImageResults(results)
	require.Len(t, got, 1)
	assert.Zero(t, got[0].Width)
	assert.Zero(t, got[0].Height)
}

func TestConvertToImageResults_PrefersStructuredImgFormat(t *testing.T) {
	results := []searxng.Result{
		{URL: "https://x/", ImgSrc: "https://x/icon.svg", ImgFormat: "SVG"},
	}
	got := convertToImageResults(results)
	require.Len(t, got, 1)
	assert.Contains(t, []string{"svg", "SVG", "image/svg"}, string(got[0].Format),
		"ImgFormat field should be respected; got %q", got[0].Format)
}

func TestConvertToImageResults_PrefersThumbnailSrcOverThumbnail(t *testing.T) {
	results := []searxng.Result{
		{URL: "https://x/", ImgSrc: "https://x/img.png",
			ThumbnailSrc: "https://thumb-cdn.example/img.png", Thumbnail: ""},
	}
	got := convertToImageResults(results)
	require.Len(t, got, 1)
	assert.Equal(t, "https://thumb-cdn.example/img.png", got[0].ThumbnailURL,
		"thumbnail_src is the actual thumbnail URL field used by most engines; thumbnail is usually empty")
}
