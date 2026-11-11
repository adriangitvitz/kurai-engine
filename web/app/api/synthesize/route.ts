import { NextResponse } from 'next/server'
import { fetchDigest, callOpenRouterJSON, coerceSynthesisFields, SourceDigest } from '@/lib/server/kurai'

const DEFAULT_MODEL = 'anthropic/claude-sonnet-5'
const MAX_URLS = 6
const MAX_OUTPUT_TOKENS = 1500

interface Body {
  query?: string
  urls?: string[]
  language?: string
}

function langName(lang: string): string {
  return lang === 'es' ? 'Spanish' : 'English'
}

function buildPrompt(query: string, sources: SourceDigest[], language: string): string {
  const blocks = sources
    .map((s, i) => `SOURCE ${i + 1} - ${s.title} (${s.url})\n${s.text}`)
    .join('\n\n---\n\n')
  return [
    `The user searched: "${query}".`,
    `Using ONLY the sources below, write a calm, neutral synthesized answer.`,
    `Write the ENTIRE response (paragraphs, fact labels, agreement, and follow-up questions) in ${langName(language)}. Keep proper nouns and numeric values as-is.`,
    `Respond with a single JSON object, no markdown fences, matching exactly:`,
    `{`,
    `  "paragraphs": string[]  // 2-3 short paragraphs, plain prose, no citations markup`,
    `  "facts": { "value": string, "label": string }[]  // 0-3 crisp key figures, e.g. {"value":"3–4 cups","label":"daily intake linked to lowest risk"}`,
    `  "agreement": string  // one short clause on how much the sources agree, e.g. "5 of 6 sources broadly agree" — empty string if unclear`,
    `  "followups": string[]  // exactly 3 natural follow-up questions`,
    `}`,
    `Do not invent facts not supported by the sources. Keep paragraphs tight and readable.`,
    `Use plain punctuation. Do not use em dashes (—); use commas, periods, or parentheses instead.`,
    ``,
    blocks,
  ].join('\n')
}

export async function POST(request: Request) {
  const apiKey = process.env.OPENROUTER_API_KEY
  if (!apiKey) {
    return NextResponse.json({ error: 'Synthesis is not configured (set OPENROUTER_API_KEY).' }, { status: 503 })
  }

  let body: Body
  try {
    body = await request.json()
  } catch {
    return NextResponse.json({ error: 'Invalid JSON body' }, { status: 400 })
  }

  const query = (body.query || '').trim()
  const urls = (body.urls || []).filter((u) => typeof u === 'string').slice(0, MAX_URLS)
  const language = body.language === 'es' ? 'es' : 'en'
  if (!query || urls.length === 0) {
    return NextResponse.json({ error: 'Missing query or urls' }, { status: 400 })
  }

  const cookie = request.headers.get('cookie') || ''
  const digests = (await Promise.all(urls.map((u) => fetchDigest(u, query, cookie)))).filter(
    (d): d is SourceDigest => d !== null
  )

  if (digests.length === 0) {
    return NextResponse.json({ error: 'Could not read any sources' }, { status: 502 })
  }

  try {
    const model = process.env.OPENROUTER_MODEL || DEFAULT_MODEL
    const system = 'You are Kurai, a careful research assistant. You only output valid JSON.'
    const raw = await callOpenRouterJSON(apiKey, model, system, buildPrompt(query, digests, language), MAX_OUTPUT_TOKENS)
    const result = coerceSynthesisFields(raw)
    if (result.paragraphs.length === 0) {
      return NextResponse.json({ error: 'Empty synthesis' }, { status: 502 })
    }
    return NextResponse.json(result)
  } catch (err) {
    console.error('Synthesize error:', err)
    return NextResponse.json({ error: 'Synthesis upstream failed' }, { status: 502 })
  }
}
