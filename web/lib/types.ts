export type ContentType =
  | 'article'
  | 'study'
  | 'video'
  | 'discussion'
  | 'guide'
  | ''

export interface Badges {
  has_ads: boolean
  tracker_heavy: boolean
  known_spam: boolean
  reputable: boolean
  content_farm: boolean
  authoritative: boolean
}

export interface SearchResult {
  url: string
  title: string
  snippet: string
  engines: string[]
  original_score: number
  adjusted_score: number
  final_score: number
  badges: Badges
  published_date?: string
  content_type?: ContentType
  primary_domain?: string
  intent_alignment?: number
  extracted_content?: string
}

export interface SearchMeta {
  total: number
  page: number
  page_size: number
  total_pages: number
  blocklist_filtered: number
  processing_time_ms: number
  from_cache: boolean
  facets?: Record<string, number>
  engines_used?: string[]
  undated_filtered?: number
  time_range?: 'day' | 'week' | 'month' | 'year'
  phrase_filtered?: number
  phrase_relaxed?: boolean
}

export interface CurrencyAnswer {
  base: string
  quote: string
  amount: number
  converted: number
  rate: number
  rate_date: string
  source: string
  minor_units: number
}

export interface ForecastPeriod {
  date: string
  min_c: number
  max_c: number
  condition?: string
}

export interface WeatherAnswer {
  place: string
  country_code?: string
  timezone?: string
  temperature_c: number
  condition?: string
  forecast?: ForecastPeriod[]
  source: string
  as_of: string
}

export interface InfoboxAnswer {
  title: string
  summary?: string
  attributes?: { label: string; value: string }[]
  links?: { title: string; url: string }[]
  image_url?: string
  source: string
}

export interface Answers {
  currency?: CurrencyAnswer
  weather?: WeatherAnswer
  infobox?: InfoboxAnswer
  corrections?: string[]
}

export interface SearchResponse {
  query: string
  results: SearchResult[]
  answers?: Answers
  meta: SearchMeta
}

export interface BookmarkVM {
  url: string
  title: string
  snippet?: string
  domain?: string
  content_type?: string
  meta?: string
  score?: number
  query?: string
  saved_at: string
}

export type BookmarkInput = Omit<BookmarkVM, 'saved_at'>

export interface BookmarkGroup {
  query?: string
  bookmarks: BookmarkVM[]
}

export interface Passage {
  heading?: string
  text: string
  score?: number
}

export interface ExtractedArticle {
  url: string
  final_url?: string
  title?: string
  byline?: string
  site_name?: string
  content_markdown?: string
  word_count?: number
}

export interface ExtractResponse {
  result?: ExtractedArticle
  passages?: Passage[]
}

export interface SynthesizeResponse {
  paragraphs: string[]
  facts: { value: string; label: string }[]
  agreement: string
  followups: string[]
}

export interface DeepResearchFact {
  n: string
  text: string
}

export interface DeepResearchStep {
  id: string
  n: string
  title: string
  summary: string
  time: string
  isStart: boolean
  desc: string
  facts: DeepResearchFact[]
}

export interface DeepResearchResponse {
  title: string
  overview: string
  metaTime: string
  metaSteps: number
  metaFacts: number
  steps: DeepResearchStep[]
  sources: SearchResult[]
  followups: string[]
}

export type DeepResearchEvent =
  | { type: 'plan'; title: string; overview: string; steps: { title: string; summary: string }[] }
  | { type: 'searching'; step: string; count: number }
  | { type: 'reading'; done: number; total: number }
  | { type: 'synthesizing' }
  | { type: 'done'; result: DeepResearchResponse }
  | { type: 'error'; message: string }

export interface HealthResponse {
  status: 'ok' | 'error'
  uptime: string
  blocklists: {
    ads_count: number
    custom_count: number
    tranco_count: number
    last_update: string
  }
}

export interface ErrorResponse {
  error: string
  message: string
}

export interface AuthRequest {
  key: string
}

export interface AuthResponse {
  user_id: string
  name: string
  expires_at: string
}
