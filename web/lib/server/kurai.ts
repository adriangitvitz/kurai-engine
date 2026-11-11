import { SearchResponse } from '../types'

export const BACKEND = process.env.KURAI_BACKEND_URL || 'http://localhost:8080'

export interface SourceDigest {
  url: string
  title: string
  text: string
}

export async function fetchDigest(url: string, focus: string, cookie: string): Promise<SourceDigest | null> {
  const params = new URLSearchParams({
    url,
    focus,
    passages_only: 'true',
    top_passages: '5',
  })
  try {
    const res = await fetch(`${BACKEND}/extract?${params}`, {
      headers: cookie ? { cookie } : {},
      cache: 'no-store',
    })
    if (!res.ok) return null
    const data = await res.json()
    const passages: { heading?: string; text?: string }[] = data.passages ?? []
    const text = passages
      .map((p) => (p.heading ? `## ${p.heading}\n` : '') + (p.text ?? ''))
      .join('\n\n')
      .slice(0, 4000)
    if (!text.trim()) return null
    return { url, title: data.result?.title || url, text }
  } catch {
    return null
  }
}

export async function fetchSearchResults(
  query: string,
  limit: number,
  cookie: string
): Promise<SearchResponse['results']> {
  const params = new URLSearchParams({ q: query, limit: String(limit) })
  try {
    const res = await fetch(`${BACKEND}/search?${params}`, {
      headers: cookie ? { cookie } : {},
      cache: 'no-store',
    })
    if (!res.ok) return []
    const data: SearchResponse = await res.json()
    return data.results ?? []
  } catch {
    return []
  }
}

const OPENROUTER_URL = 'https://openrouter.ai/api/v1/chat/completions'

export function extractJson(content: string): unknown {
  try {
    return JSON.parse(content)
  } catch {
    const start = content.indexOf('{')
    const end = content.lastIndexOf('}')
    if (start >= 0 && end > start) {
      try {
        return JSON.parse(content.slice(start, end + 1))
      } catch {
      }
    }
    return {}
  }
}

export async function callOpenRouterJSON(
  apiKey: string,
  model: string,
  system: string,
  user: string,
  maxTokens: number
): Promise<unknown> {
  const res = await fetch(OPENROUTER_URL, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
      'X-Title': 'Kurai',
    },
    body: JSON.stringify({
      model,
      temperature: 0.3,
      max_tokens: maxTokens,
      response_format: { type: 'json_object' },
      messages: [
        { role: 'system', content: system },
        { role: 'user', content: user },
      ],
    }),
  })
  if (!res.ok) {
    const detail = await res.text().catch(() => '')
    throw new Error(`OpenRouter ${res.status}: ${detail.slice(0, 300)}`)
  }
  const data = await res.json()
  const content: string = data?.choices?.[0]?.message?.content ?? ''
  return extractJson(content)
}

export interface SynthesisFields {
  paragraphs: string[]
  facts: { value: string; label: string }[]
  agreement: string
  followups: string[]
}

export function coerceSynthesisFields(
  raw: unknown,
  caps: { maxParagraphs: number; maxFacts: number } = { maxParagraphs: 4, maxFacts: 3 }
): SynthesisFields {
  const o = (raw ?? {}) as Record<string, unknown>
  const strArr = (v: unknown): string[] =>
    Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : []
  const facts = Array.isArray(o.facts)
    ? (o.facts as unknown[])
        .map((f) => {
          const ff = (f ?? {}) as Record<string, unknown>
          return { value: String(ff.value ?? ''), label: String(ff.label ?? '') }
        })
        .filter((f) => f.value && f.label)
        .slice(0, caps.maxFacts)
    : []
  return {
    paragraphs: strArr(o.paragraphs).slice(0, caps.maxParagraphs),
    facts,
    agreement: typeof o.agreement === 'string' ? o.agreement : '',
    followups: strArr(o.followups).slice(0, 3),
  }
}
