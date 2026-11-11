package ranker

import (
	"regexp"
	"strings"

	"github.com/adriangitvitz/kurai/pkg/urlutil"
)

type ImageFormat string

const (
	ImageFormatWebP ImageFormat = "webp"
	ImageFormatPNG  ImageFormat = "png"
	ImageFormatJPEG ImageFormat = "jpeg"
	ImageFormatJPG  ImageFormat = "jpg"
	ImageFormatGIF  ImageFormat = "gif"
	ImageFormatSVG  ImageFormat = "svg"
	ImageFormatAVIF ImageFormat = "avif"
)

type ImageSize string

const (
	ImageSizeSmall  ImageSize = "small"
	ImageSizeMedium ImageSize = "medium"
	ImageSizeLarge  ImageSize = "large"
)

type ImageResult struct {
	URL          string      `json:"url"`
	ThumbnailURL string      `json:"thumbnail_url"`
	Title        string      `json:"title"`
	AltText      string      `json:"alt_text"`
	Source       string      `json:"source"`
	SourceDomain string      `json:"source_domain"`
	Width        int         `json:"width"`
	Height       int         `json:"height"`
	Format       ImageFormat `json:"format"`
	FileSize     int64       `json:"file_size"`
	Engines      []string    `json:"engines"`

	RelevanceScore float64 `json:"relevance_score"`
	QualityScore   float64 `json:"quality_score"`
	AuthorityScore float64 `json:"authority_score"`
	FinalScore     float64 `json:"final_score"`
}

type ImageRankingConfig struct {
	RelevanceWeight float64 `yaml:"relevance_weight"`
	QualityWeight   float64 `yaml:"quality_weight"`
	AuthorityWeight float64 `yaml:"authority_weight"`

	PreferredFormats []ImageFormat `yaml:"preferred_formats"`
	MinWidth         int           `yaml:"min_width"`
	MinHeight        int           `yaml:"min_height"`
	PreferHighRes    bool          `yaml:"prefer_high_res"`
}

func DefaultImageRankingConfig() ImageRankingConfig {
	return ImageRankingConfig{
		RelevanceWeight: 0.5,
		QualityWeight:   0.3,
		AuthorityWeight: 0.2,
		PreferredFormats: []ImageFormat{
			ImageFormatWebP, ImageFormatAVIF, ImageFormatPNG, ImageFormatJPEG,
		},
		MinWidth:      200,
		MinHeight:     200,
		PreferHighRes: true,
	}
}

type ImageRanker struct {
	config       ImageRankingConfig
	blocklist    BlocklistChecker
	formatScores map[ImageFormat]float64
}

func NewImageRanker(config ImageRankingConfig, blocklist BlocklistChecker) *ImageRanker {
	ranker := &ImageRanker{
		config:    config,
		blocklist: blocklist,
		formatScores: map[ImageFormat]float64{
			ImageFormatAVIF: 1.0,
			ImageFormatWebP: 0.95,
			ImageFormatPNG:  0.85,
			ImageFormatJPEG: 0.80,
			ImageFormatJPG:  0.80,
			ImageFormatSVG:  0.75,
			ImageFormatGIF:  0.60,
		},
	}

	return ranker
}

func (ir *ImageRanker) ScoreImageRelevance(queryTerms []string, img ImageResult) float64 {
	if len(queryTerms) == 0 {
		return 0.5
	}

	textToCheck := strings.ToLower(img.Title + " " + img.AltText)

	matchCount := 0
	for _, term := range queryTerms {
		termLower := strings.ToLower(term)
		if strings.Contains(textToCheck, termLower) {
			matchCount++
		}
	}

	matchPercent := float64(matchCount) / float64(len(queryTerms))

	queryLower := strings.ToLower(strings.Join(queryTerms, " "))
	if strings.Contains(textToCheck, queryLower) {
		matchPercent = min(matchPercent*1.3, 1.0)
	}

	return matchPercent
}

func (ir *ImageRanker) ScoreResolution(img ImageResult) float64 {
	width := img.Width
	height := img.Height

	if width < ir.config.MinWidth || height < ir.config.MinHeight {
		return 0.1
	}

	pixels := width * height

	switch {
	case pixels >= 4000000:
		return 1.0
	case pixels >= 2000000:
		return 0.9
	case pixels >= 1000000:
		return 0.8
	case pixels >= 500000:
		return 0.7
	case pixels >= 250000:
		return 0.6
	case pixels >= 100000:
		return 0.5
	default:
		return 0.3
	}
}

func (ir *ImageRanker) ScoreImageSourceAuthority(img ImageResult) float64 {
	domain := img.SourceDomain
	if domain == "" {
		domain = urlutil.ExtractDomain(img.Source)
	}

	if ir.blocklist != nil && ir.blocklist.IsBlocked(domain) {
		return 0.0
	}

	if ir.blocklist != nil {
		trancoRank := ir.blocklist.GetTrancoRank(domain)
		if trancoRank > 0 {
			switch {
			case trancoRank <= 1000:
				return 1.0
			case trancoRank <= 10000:
				return 0.9
			case trancoRank <= 100000:
				return 0.8
			case trancoRank <= 1000000:
				return 0.7
			}
		}
	}

	qualitySources := map[string]float64{
		"unsplash.com":          0.95,
		"pexels.com":            0.90,
		"flickr.com":            0.85,
		"wikipedia.org":         0.90,
		"wikimedia.org":         0.90,
		"commons.wikimedia.org": 0.90,
		"deviantart.com":        0.80,
		"500px.com":             0.85,
		"shutterstock.com":      0.85,
		"gettyimages.com":       0.85,
		"istockphoto.com":       0.85,
		"adobe.com":             0.90,
		"dribbble.com":          0.85,
		"behance.net":           0.85,
	}

	if score, ok := qualitySources[domain]; ok {
		return score
	}

	return 0.5
}

func (ir *ImageRanker) ScoreFormatPreference(img ImageResult) float64 {
	if score, ok := ir.formatScores[img.Format]; ok {
		return score
	}

	format := DetectImageFormat(img.URL)
	if score, ok := ir.formatScores[format]; ok {
		return score
	}

	return 0.5
}

func DetectImageFormat(url string) ImageFormat {
	urlLower := strings.ToLower(url)

	formatPatterns := []struct {
		pattern string
		format  ImageFormat
	}{
		{".webp", ImageFormatWebP},
		{".avif", ImageFormatAVIF},
		{".png", ImageFormatPNG},
		{".jpg", ImageFormatJPG},
		{".jpeg", ImageFormatJPEG},
		{".gif", ImageFormatGIF},
		{".svg", ImageFormatSVG},
		{"format=webp", ImageFormatWebP},
		{"format=png", ImageFormatPNG},
		{"format=jpg", ImageFormatJPG},
		{"format=jpeg", ImageFormatJPEG},
		{"fm=webp", ImageFormatWebP},
		{"fm=png", ImageFormatPNG},
		{"fm=jpg", ImageFormatJPG},
	}

	for _, p := range formatPatterns {
		if strings.Contains(urlLower, p.pattern) {
			return p.format
		}
	}

	return ""
}

func (ir *ImageRanker) RankImages(queryTerms []string, images []ImageResult) []ImageResult {
	for i := range images {
		relevance := ir.ScoreImageRelevance(queryTerms, images[i])
		quality := ir.ScoreResolution(images[i])
		authority := ir.ScoreImageSourceAuthority(images[i])
		format := ir.ScoreFormatPreference(images[i])

		qualityCombined := (quality*0.7 + format*0.3)

		images[i].RelevanceScore = relevance
		images[i].QualityScore = qualityCombined
		images[i].AuthorityScore = authority

		images[i].FinalScore = relevance*ir.config.RelevanceWeight +
			qualityCombined*ir.config.QualityWeight +
			authority*ir.config.AuthorityWeight
	}

	sortImagesByScore(images)

	return images
}

func sortImagesByScore(images []ImageResult) {
	for i := 1; i < len(images); i++ {
		for j := i; j > 0 && images[j].FinalScore > images[j-1].FinalScore; j-- {
			images[j], images[j-1] = images[j-1], images[j]
		}
	}
}

func (ir *ImageRanker) FilterBySize(images []ImageResult, size ImageSize) []ImageResult {
	var filtered []ImageResult

	for _, img := range images {
		pixels := img.Width * img.Height
		var matchesSize bool

		switch size {
		case ImageSizeSmall:
			matchesSize = pixels < 160000
		case ImageSizeMedium:
			matchesSize = pixels >= 160000 && pixels < 1440000
		case ImageSizeLarge:
			matchesSize = pixels >= 1440000
		default:
			matchesSize = true
		}

		if matchesSize {
			filtered = append(filtered, img)
		}
	}

	return filtered
}

func (ir *ImageRanker) FilterByFormat(images []ImageResult, formats []ImageFormat) []ImageResult {
	if len(formats) == 0 {
		return images
	}

	formatSet := make(map[ImageFormat]bool)
	for _, f := range formats {
		formatSet[f] = true
	}

	var filtered []ImageResult
	for _, img := range images {
		format := img.Format
		if format == "" {
			format = DetectImageFormat(img.URL)
		}
		if formatSet[format] {
			filtered = append(filtered, img)
		}
	}

	return filtered
}

type ImageSearchOptions struct {
	SafeSearch int           `json:"safesearch"`
	Size       ImageSize     `json:"size"`
	Formats    []ImageFormat `json:"formats"`
	MinWidth   int           `json:"min_width"`
	MinHeight  int           `json:"min_height"`
}

func DefaultImageSearchOptions() ImageSearchOptions {
	return ImageSearchOptions{
		SafeSearch: 1,
		MinWidth:   100,
		MinHeight:  100,
	}
}

func ValidateSafeSearch(value int) int {
	if value < 0 {
		return 0
	}
	if value > 2 {
		return 2
	}
	return value
}

func ParseImageSize(s string) ImageSize {
	switch strings.ToLower(s) {
	case "small", "s":
		return ImageSizeSmall
	case "medium", "m", "med":
		return ImageSizeMedium
	case "large", "l", "lg":
		return ImageSizeLarge
	default:
		return ""
	}
}

func IsSafeImageURL(url string) bool {
	suspiciousPatterns := []string{
		"javascript:",
		"data:",
		"vbscript:",
		"file://",
		"ftp://",
		"%00",
		"<script",
		"onclick",
		"onerror",
	}

	urlLower := strings.ToLower(url)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(urlLower, pattern) {
			return false
		}
	}

	if !strings.HasPrefix(urlLower, "http://") && !strings.HasPrefix(urlLower, "https://") {
		return false
	}

	return true
}

var aspectRatioPattern = regexp.MustCompile(`(\d+)[x\-:](\d+)`)

func ExtractAspectRatio(img ImageResult) float64 {
	if img.Width > 0 && img.Height > 0 {
		return float64(img.Width) / float64(img.Height)
	}

	matches := aspectRatioPattern.FindStringSubmatch(img.URL)
	if len(matches) == 3 {
		return 0
	}

	return 0
}
