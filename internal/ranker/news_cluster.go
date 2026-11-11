package ranker

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

type StoryCluster struct {
	ID             string       `json:"id"`
	MainStory      *NewsResult  `json:"main_story"`
	RelatedStories []NewsResult `json:"related_stories"`
	SourceCount    int          `json:"source_count"`
	IsTrending     bool         `json:"is_trending"`
	Keywords       []string     `json:"keywords,omitempty"`
}

type ClusterConfig struct {
	SimilarityThreshold float64 `yaml:"similarity_threshold"`
	MinClusterSize      int     `yaml:"min_cluster_size"`
	MaxClustersReturned int     `yaml:"max_clusters_returned"`
}

func DefaultClusterConfig() ClusterConfig {
	return ClusterConfig{
		SimilarityThreshold: 0.6,
		MinClusterSize:      3,
		MaxClustersReturned: 10,
	}
}

type NewsClusterer struct {
	config ClusterConfig
}

func NewNewsClusterer(config ClusterConfig) *NewsClusterer {
	return &NewsClusterer{config: config}
}

func (nc *NewsClusterer) ClusterStories(news []NewsResult) []StoryCluster {
	if len(news) == 0 {
		return nil
	}

	storyTokens := make([]map[string]bool, len(news))
	for i, n := range news {
		storyTokens[i] = nc.extractTokens(n.Title)
	}

	clustered := make([]bool, len(news))
	var clusters []StoryCluster

	for i := 0; i < len(news); i++ {
		if clustered[i] {
			continue
		}

		cluster := StoryCluster{
			ID:             generateClusterID(news[i].Title),
			MainStory:      &news[i],
			RelatedStories: []NewsResult{},
			Keywords:       extractKeywords(news[i].Title),
		}

		clustered[i] = true
		sourceSet := map[string]bool{news[i].SourceDomain: true}

		for j := i + 1; j < len(news); j++ {
			if clustered[j] {
				continue
			}

			similarity := nc.jaccardSimilarity(storyTokens[i], storyTokens[j])

			if similarity >= nc.config.SimilarityThreshold {
				cluster.RelatedStories = append(cluster.RelatedStories, news[j])
				sourceSet[news[j].SourceDomain] = true
				clustered[j] = true
				news[j].ClusterID = cluster.ID
			}
		}

		cluster.SourceCount = len(sourceSet)
		cluster.IsTrending = cluster.SourceCount >= nc.config.MinClusterSize

		news[i].ClusterID = cluster.ID
		news[i].IsMainStory = true

		clusters = append(clusters, cluster)
	}

	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].IsTrending != clusters[j].IsTrending {
			return clusters[i].IsTrending
		}
		if clusters[i].SourceCount != clusters[j].SourceCount {
			return clusters[i].SourceCount > clusters[j].SourceCount
		}
		return clusters[i].MainStory.FinalScore > clusters[j].MainStory.FinalScore
	})

	if len(clusters) > nc.config.MaxClustersReturned {
		clusters = clusters[:nc.config.MaxClustersReturned]
	}

	return clusters
}

func (nc *NewsClusterer) extractTokens(text string) map[string]bool {
	tokens := make(map[string]bool)

	text = strings.ToLower(text)

	var cleanText strings.Builder
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			cleanText.WriteRune(r)
		} else if r == '-' || r == '\'' {
			cleanText.WriteRune(' ')
		}
	}

	words := strings.Fields(cleanText.String())

	for _, word := range words {
		if len(word) > 2 && !isStopWord(word) {
			tokens[word] = true
		}
	}

	return tokens
}

func (nc *NewsClusterer) jaccardSimilarity(set1, set2 map[string]bool) float64 {
	if len(set1) == 0 || len(set2) == 0 {
		return 0
	}

	intersection := 0
	for token := range set1 {
		if set2[token] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection

	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true, "was": true,
		"are": true, "were": true, "been": true, "be": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "must": true,
		"it": true, "its": true, "this": true, "that": true, "these": true, "those": true,
		"he": true, "she": true, "they": true, "we": true, "you": true, "who": true,
		"what": true, "when": true, "where": true, "why": true, "how": true,
		"new": true, "says": true, "said": true, "after": true, "over": true, "into": true,
		"about": true, "more": true, "some": true, "all": true, "than": true, "other": true,
	}

	return stopWords[word]
}

func generateClusterID(title string) string {
	h := sha256.Sum256([]byte(strings.ToLower(title)))
	return hex.EncodeToString(h[:8])
}

func extractKeywords(title string) []string {
	text := strings.ToLower(title)

	var cleanText strings.Builder
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			cleanText.WriteRune(r)
		}
	}

	words := strings.Fields(cleanText.String())

	var keywords []string
	for _, word := range words {
		if len(word) > 3 && !isStopWord(word) {
			keywords = append(keywords, word)
		}
	}

	if len(keywords) > 5 {
		keywords = keywords[:5]
	}

	return keywords
}

func (nc *NewsClusterer) GetTrendingStories(news []NewsResult) []NewsResult {
	clusters := nc.ClusterStories(news)

	var trending []NewsResult
	for _, cluster := range clusters {
		if cluster.IsTrending && cluster.MainStory != nil {
			trending = append(trending, *cluster.MainStory)
		}
	}

	return trending
}

func (nc *NewsClusterer) DeduplicateWithClustering(news []NewsResult) []NewsResult {
	clusters := nc.ClusterStories(news)

	var deduplicated []NewsResult
	for _, cluster := range clusters {
		if cluster.MainStory != nil {
			deduplicated = append(deduplicated, *cluster.MainStory)
		}
	}

	return deduplicated
}

func (nc *NewsClusterer) FindRelatedStories(target NewsResult, candidates []NewsResult, limit int) []NewsResult {
	targetTokens := nc.extractTokens(target.Title)

	type scoredResult struct {
		result     NewsResult
		similarity float64
	}

	var scored []scoredResult

	for _, candidate := range candidates {
		if candidate.URL == target.URL {
			continue
		}

		candidateTokens := nc.extractTokens(candidate.Title)
		similarity := nc.jaccardSimilarity(targetTokens, candidateTokens)

		if similarity > 0.3 {
			scored = append(scored, scoredResult{
				result:     candidate,
				similarity: similarity,
			})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].similarity > scored[j].similarity
	})

	var related []NewsResult
	for i := 0; i < len(scored) && i < limit; i++ {
		related = append(related, scored[i].result)
	}

	return related
}

func (nc *NewsClusterer) MergeClusterResults(clusters []StoryCluster) []NewsResult {
	var merged []NewsResult

	for _, cluster := range clusters {
		if cluster.MainStory != nil {
			merged = append(merged, *cluster.MainStory)
		}
	}

	for _, cluster := range clusters {
		for _, related := range cluster.RelatedStories {
			merged = append(merged, related)
		}
	}

	return merged
}

func (nc *NewsClusterer) SetConfig(config ClusterConfig) {
	nc.config = config
}
