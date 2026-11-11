import { SearchResult, ContentType } from './types'
import { getHostname, formatTimeAgo } from './api'

export type CardSize = 'feature' | 'medium' | 'compact'

export interface SourceVM {
  id: string
  url: string
  title: string
  snippet: string
  type: string
  rawType: ContentType
  domain: string
  meta: string
  score: number
  size: CardSize
  hasImg: boolean
  imgLabel: string
  query?: string
  savedAt?: string
}

const TYPE_LABEL: Record<string, string> = {
  article: 'Article',
  study: 'Study',
  video: 'Video',
  discussion: 'Discussion',
  guide: 'Guide',
}

export function contentTypeLabel(t?: ContentType): string {
  return (t && TYPE_LABEL[t]) || 'Result'
}

export function cardSize(rank: number): CardSize {
  if (rank === 0) return 'feature'
  if (rank <= 2) return 'medium'
  return 'compact'
}

export function metaLine(r: SearchResult): string {
  const ago = r.published_date ? formatTimeAgo(r.published_date) : ''
  if (ago) return ago
  const n = r.engines?.length ?? 0
  return n ? `${n} engine${n > 1 ? 's' : ''}` : ''
}

export function toSource(r: SearchResult, rank: number): SourceVM {
  const size = cardSize(rank)
  return {
    id: r.url,
    url: r.url,
    title: r.title,
    snippet: r.snippet,
    rawType: r.content_type ?? '',
    type: contentTypeLabel(r.content_type),
    domain: getHostname(r.url),
    meta: metaLine(r),
    score: r.final_score ?? 0,
    size,
    hasImg: size !== 'compact',
    imgLabel: contentTypeLabel(r.content_type).toLowerCase(),
  }
}

const FACET_TYPE: Record<string, string> = {
  Articles: 'article',
  Studies: 'study',
  Videos: 'video',
  Discussions: 'discussion',
  Guides: 'guide',
}
const TYPE_FACET: Record<string, string> = {
  article: 'Articles',
  study: 'Studies',
  video: 'Videos',
  discussion: 'Discussions',
  guide: 'Guides',
}
const TYPE_ORDER = ['article', 'study', 'video', 'discussion', 'guide']

export function facetKeyToContentType(key: string): string {
  return FACET_TYPE[key] ?? ''
}

export interface Facet {
  key: string
  label: string
  count: number
}

export function facetModelFromCounts(counts: Record<string, number> | undefined): Facet[] {
  const c = counts ?? {}
  const all = TYPE_ORDER.reduce((sum, t) => sum + (c[t] ?? 0), 0)
  const facets: Facet[] = [{ key: 'All', label: 'All', count: all }]
  for (const t of TYPE_ORDER) {
    if (c[t]) facets.push({ key: TYPE_FACET[t], label: TYPE_FACET[t], count: c[t] })
  }
  return facets
}

export function topUrls(sources: SourceVM[], n = 5): string[] {
  return sources.slice(0, n).map((s) => s.url)
}
