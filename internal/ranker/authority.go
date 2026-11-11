package ranker

import (
	"github.com/adriangitvitz/kurai/internal/config"
)

type AuthorityCategory int

const (
	CategoryNone AuthorityCategory = iota
	CategoryOfficialDocs
	CategoryCodeRepository
	CategoryDeveloperQA
	CategoryQualityBlog
	CategoryPaperArchive
	CategoryAcademicPublisher
	CategoryWikipedia
	CategoryMajorNews
	CategoryQualityNews
)

func GetAuthorityCategory(domain string) AuthorityCategory {
	return CategoryNone
}

func AuthorityBonus(domain string, cfg *config.AuthorityConfig) int {
	return 0
}

func AuthorityBonusForCategory(cat AuthorityCategory, cfg *config.AuthorityConfig) int {
	return 0
}

func AuthorityBonusWithIntent(domain string, queryIntent QueryCategory, cfg *config.AuthorityConfig) int {
	return 0
}

func QueryDomainMatchBonus(queryTerms []string, domain string, cfg *config.AuthorityConfig) int {
	return 0
}
