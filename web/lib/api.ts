import {
  SearchResponse,
  HealthResponse,
  AuthResponse,
  ExtractResponse,
  ExtractedArticle,
  Passage,
  SynthesizeResponse,
  DeepResearchEvent,
  BookmarkVM,
  BookmarkInput,
  BookmarkGroup,
} from './types'
import type { SearchFilters } from './filters'

export type {
  SearchResponse,
  HealthResponse,
  AuthResponse,
  ExtractResponse,
  Passage,
  SynthesizeResponse,
  DeepResearchEvent,
  BookmarkVM,
  BookmarkInput,
  BookmarkGroup,
}

const API_BASE = '/api'

function redirectToLogin() {
  if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
    const here = window.location.pathname + window.location.search
    window.location.href = here === '/' ? '/login' : `/login?next=${encodeURIComponent(here)}`
  }
}

export class APIError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: string
  ) {
    super(message)
    this.name = 'APIError'
  }
}

export interface SearchOpts {
  page?: number
  limit?: number
  debug?: boolean
  contentType?: string
  filters?: SearchFilters
}

export async function search(query: string, opts: SearchOpts = {}): Promise<SearchResponse> {
  const params = new URLSearchParams({ q: query, page: String(opts.page ?? 1) })
  if (opts.limit) params.set('limit', String(opts.limit))
  if (opts.debug) params.set('debug', 'true')
  if (opts.contentType) params.set('content_type', opts.contentType)
  if (opts.filters?.sites.length) params.set('include_domains', opts.filters.sites.join(','))
  if (opts.filters?.from) params.set('date_from', opts.filters.from)
  if (opts.filters?.to) params.set('date_to', opts.filters.to)

  const res = await fetch(`${API_BASE}/search?${params}`, {
    cache: 'no-store',
    credentials: 'include',
  })

  if (!res.ok) {
    if (res.status === 401) redirectToLogin()
    const error = await res.json().catch(() => ({ message: 'Search failed' }))
    throw new APIError(error.message || 'Search failed', res.status)
  }

  return res.json()
}

export async function listBookmarks(): Promise<BookmarkVM[]> {
  const res = await fetch(`${API_BASE}/bookmarks`, { cache: 'no-store', credentials: 'include' })
  if (!res.ok) {
    if (res.status === 401) redirectToLogin()
    throw new APIError('Failed to load the board', res.status)
  }
  const data = (await res.json()) as { bookmarks?: BookmarkVM[] }
  return data.bookmarks ?? []
}

export async function listBookmarkGroups(): Promise<BookmarkGroup[]> {
  const res = await fetch(`${API_BASE}/bookmarks?group=query`, {
    cache: 'no-store',
    credentials: 'include',
  })
  if (!res.ok) {
    if (res.status === 401) redirectToLogin()
    throw new APIError('Failed to load the board', res.status)
  }
  const data = (await res.json()) as { groups?: BookmarkGroup[] }
  return data.groups ?? []
}

export async function saveBookmark(b: BookmarkInput): Promise<BookmarkVM> {
  const res = await fetch(`${API_BASE}/bookmarks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify(b),
  })
  if (!res.ok) {
    if (res.status === 401) redirectToLogin()
    throw new APIError('Failed to save the bookmark', res.status)
  }
  return res.json()
}

export async function deleteBookmark(url: string): Promise<void> {
  const res = await fetch(`${API_BASE}/bookmarks?url=${encodeURIComponent(url)}`, {
    method: 'DELETE',
    credentials: 'include',
  })
  if (!res.ok) {
    if (res.status === 401) redirectToLogin()
    throw new APIError('Failed to delete the bookmark', res.status)
  }
}

export async function extractPassages(url: string, focus: string, top = 4): Promise<Passage[]> {
  const params = new URLSearchParams({
    url,
    focus,
    passages_only: 'true',
    top_passages: String(top),
  })

  try {
    const res = await fetch(`${API_BASE}/extract?${params}`, {
      cache: 'no-store',
      credentials: 'include',
    })
    if (!res.ok) {
      if (res.status === 401) redirectToLogin()
      return []
    }
    const data: ExtractResponse = await res.json()
    return data.passages ?? []
  } catch {
    return []
  }
}

export async function extractPage(url: string): Promise<ExtractedArticle> {
  const params = new URLSearchParams({ url, format: 'markdown' })
  const res = await fetch(`${API_BASE}/extract?${params}`, { cache: 'no-store', credentials: 'include' })
  if (!res.ok) {
    if (res.status === 401) redirectToLogin()
    const body = await res.json().catch(() => ({}))
    throw new APIError(body.error?.message || 'Extraction failed', res.status, body.error?.code)
  }
  const data: ExtractResponse = await res.json()
  if (!data.result?.content_markdown) throw new APIError('No readable content', 422, 'extraction_failed')
  return data.result
}

export async function synthesize(
  query: string,
  urls: string[],
  language: 'en' | 'es' = 'en'
): Promise<SynthesizeResponse> {
  const res = await fetch('/api/synthesize', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ query, urls, language }),
  })

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: 'Synthesis failed' }))
    throw new APIError(error.error || 'Synthesis failed', res.status)
  }

  return res.json()
}

export async function runDeepResearch(
  query: string,
  language: 'en' | 'es',
  onEvent: (event: DeepResearchEvent) => void
): Promise<void> {
  const res = await fetch('/api/deep-research', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ query, language }),
  })

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: 'Deep research failed' }))
    throw new APIError(error.error || 'Deep research failed', res.status)
  }
  if (!res.body) {
    throw new APIError('Deep research stream unavailable', res.status)
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })

    let sep: number
    while ((sep = buffer.indexOf('\n\n')) !== -1) {
      const frame = buffer.slice(0, sep)
      buffer = buffer.slice(sep + 2)
      const line = frame.split('\n').find((l) => l.startsWith('data: '))
      if (!line) continue
      try {
        onEvent(JSON.parse(line.slice('data: '.length)) as DeepResearchEvent)
      } catch {
      }
    }
  }
}

export async function health(): Promise<HealthResponse> {
  const res = await fetch(`${API_BASE}/health`, { next: { revalidate: 60 } })
  if (!res.ok) throw new APIError('Health check failed', res.status)
  return res.json()
}

export async function authenticate(key: string): Promise<AuthResponse> {
  const res = await fetch(`${API_BASE}/auth`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ key }),
  })

  if (!res.ok) {
    const error = await res.json().catch(() => ({ message: 'Authentication failed' }))
    throw new APIError(error.message || 'Authentication failed', res.status)
  }

  return res.json()
}

export function getHostname(url: string): string {
  try {
    return new URL(url).hostname.replace(/^www\./, '')
  } catch {
    return url
  }
}

export function formatTimeAgo(dateStr: string): string {
  if (!dateStr) return ''

  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) return ''

  const diffMs = Date.now() - date.getTime()
  const diffMin = Math.floor(diffMs / 60000)
  const diffHour = Math.floor(diffMin / 60)
  const diffDay = Math.floor(diffHour / 24)

  if (diffMin < 1) return 'just now'
  if (diffMin < 60) return `${diffMin}m ago`
  if (diffHour < 24) return `${diffHour}h ago`
  if (diffDay < 7) return `${diffDay}d ago`
  if (diffDay < 30) return `${Math.floor(diffDay / 7)}w ago`
  return `${Math.floor(diffDay / 30)}mo ago`
}
