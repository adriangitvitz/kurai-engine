package ranker

import "testing"

func TestClassifyContentType(t *testing.T) {
	cases := []struct {
		name  string
		url   string
		score DomainScores
		want  ContentType
	}{
		{"youtube", "https://www.youtube.com/watch?v=abc", nil, ContentVideo},
		{"youtu.be", "https://youtu.be/abc", nil, ContentVideo},
		{"vimeo", "https://vimeo.com/12345", nil, ContentVideo},
		{"reddit", "https://www.reddit.com/r/golang/comments/x", nil, ContentDiscussion},
		{"hackernews", "https://news.ycombinator.com/item?id=1", nil, ContentDiscussion},
		{"stackoverflow", "https://stackoverflow.com/questions/1/x", nil, ContentDiscussion},
		{"forum path", "https://example.com/forum/thread-42", nil, ContentDiscussion},
		{"arxiv", "https://arxiv.org/abs/2401.00001", nil, ContentStudy},
		{"edu", "https://cs.stanford.edu/research/x", nil, ContentStudy},
		{"docs path", "https://example.com/docs/install", nil, ContentGuide},
		{"mdn", "https://developer.mozilla.org/en-US/docs/Web", nil, ContentGuide},
		{"tutorial path", "https://example.com/tutorial/getting-started", nil, ContentGuide},
		{"plain article", "https://example.com/blog/post", nil, ContentArticle},
		{"academic vocab fallback", "https://example.com/x", DomainScores{DomainAcademic: 1.0}, ContentStudy},
		{"documentation vocab fallback", "https://example.com/x", DomainScores{DomainDocumentation: 1.0}, ContentGuide},
		{"video beats vocab", "https://youtube.com/watch", DomainScores{DomainAcademic: 1.0}, ContentVideo},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyContentType(tc.url, tc.score)
			if got != tc.want {
				t.Errorf("ClassifyContentType(%q) = %q, want %q", tc.url, got, tc.want)
			}
		})
	}
}
