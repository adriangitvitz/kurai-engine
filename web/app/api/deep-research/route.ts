import { NextResponse } from 'next/server'
import { fetchDigest, fetchSearchResults, callOpenRouterJSON, SourceDigest } from '@/lib/server/kurai'
import type { DeepResearchEvent, DeepResearchStep, DeepResearchFact } from '@/lib/types'
import type { SearchResult } from '@/lib/types'

export const dynamic = 'force-dynamic'

const MAX_STEPS = 5
const MAX_SOURCES_PER_STEP = 3

const DEFAULT_SYNTH_MODEL = 'anthropic/claude-sonnet-5'
const DEFAULT_FAST_MODEL = 'anthropic/claude-haiku-4.5'

const PLAN_MAX_TOKENS = 600
const SUMMARY_MAX_TOKENS = 300
const ASSEMBLE_MAX_TOKENS = 1800

const WORDS_PER_MINUTE = 200
const MIN_STEP_MINUTES = 3

interface Body {
  query?: string
  language?: string
}

interface PlannedStep {
  title: string
  summary: string
}

interface StepCandidate {
  result: SearchResult
  step: string
}

interface StepFact {
  step: string
  facts: string[]
}

function langName(lang: string): string {
  return lang === 'es' ? 'Spanish' : 'English'
}

function floorVariant(model: string): string {
  return model.includes(':') ? model : `${model}:floor`
}

function wordCount(s: string): number {
  return s.trim().split(/\s+/).filter(Boolean).length
}

function estimateStepTime(desc: string, facts: string[]): string {
  const words = wordCount(desc) + facts.reduce((n, f) => n + wordCount(f), 0)
  const minutes = Math.max(MIN_STEP_MINUTES, Math.round(words / WORDS_PER_MINUTE))
  return `${minutes} min`
}

function totalReadTime(steps: DeepResearchStep[]): string {
  const total = steps.reduce((sum, s) => sum + (parseInt(s.time, 10) || 0), 0)
  return `~${total} min read`
}

function planPrompt(query: string): string {
  return [
    `The user wants to learn about: "${query}".`,
    `Design a short, ordered learning path that teaches this topic from the ground up: simplest ideas first, building toward practical application. Break it into 3-${MAX_STEPS} sequential steps.`,
    `Also write a clean, short topic title (2-5 words, a noun phrase, not the literal query) and a one-paragraph overview (2-3 sentences) of what the topic is and why it's worth learning.`,
    `Respond with a single JSON object, no markdown fences: {"title": string, "overview": string, "steps": [{"title": string, "summary": string}]}.`,
    `Each step's "title" is a short phrase (e.g. "The building blocks"); "summary" is a single clause describing it (e.g. "Servers, databases, caches, load balancers.").`,
  ].join('\n')
}

function summarizePrompt(topicTitle: string, stepTitle: string, digest: SourceDigest): string {
  return [
    `Step: "${stepTitle}" — part of learning about "${topicTitle}".`,
    ``,
    `Source: ${digest.title} (${digest.url})`,
    digest.text,
    ``,
    `Extract 2-4 atomic facts from this source relevant to this step: single-idea, self-contained statements someone could learn on their own (each under 25 words). Only include facts actually supported by the text; do not invent anything.`,
    `Respond with a single JSON object, no markdown fences: {"facts": string[]}. If nothing in the source is relevant, respond {"facts": []}.`,
  ].join('\n')
}

function assemblePrompt(topicTitle: string, overview: string, steps: PlannedStep[], stepFacts: StepFact[], language: string): string {
  const blocks = steps
    .map((st, i) => {
      const raw = stepFacts.find((sf) => sf.step === st.title)?.facts ?? []
      const rows = raw.length > 0 ? raw.map((f) => `- ${f}`).join('\n') : '(no sources found for this step)'
      return `STEP ${i + 1}: "${st.title}" — ${st.summary}\nRaw facts:\n${rows}`
    })
    .join('\n\n')

  return [
    `The user is learning about "${topicTitle}": ${overview}`,
    ``,
    `Below are raw facts gathered per step from real sources. For each step, write:`,
    `- "desc": a short paragraph (2-4 sentences) explaining the step clearly, in the context of the overall topic`,
    `- "facts": 3-5 clean, atomic facts curated from the raw list for that step (merge near-duplicates, drop anything unsupported, each under 25 words)`,
    `Also write exactly 3 natural follow-up questions a learner might ask next.`,
    `Write the ENTIRE response in ${langName(language)}. Keep proper nouns and numeric values as-is. Use plain punctuation — do not use em dashes (—); use commas, periods, or parentheses instead.`,
    `Respond with a single JSON object, no markdown fences, matching exactly:`,
    `{`,
    `  "steps": [{"desc": string, "facts": string[]}]  // same order as below, one entry per step`,
    `  "followups": string[]  // exactly 3`,
    `}`,
    `Do not invent facts not supported by the raw lists below.`,
    ``,
    blocks,
  ].join('\n')
}

export async function POST(request: Request) {
  const apiKey = process.env.OPENROUTER_API_KEY
  if (!apiKey) {
    return NextResponse.json({ error: 'Deep research is not configured (set OPENROUTER_API_KEY).' }, { status: 503 })
  }

  let body: Body
  try {
    body = await request.json()
  } catch {
    return NextResponse.json({ error: 'Invalid JSON body' }, { status: 400 })
  }

  const query = (body.query || '').trim()
  const language = body.language === 'es' ? 'es' : 'en'
  if (!query) {
    return NextResponse.json({ error: 'Missing query' }, { status: 400 })
  }

  const cookie = request.headers.get('cookie') || ''
  const fastModel = process.env.OPENROUTER_MODEL_FAST || DEFAULT_FAST_MODEL
  const synthModel = process.env.OPENROUTER_MODEL || DEFAULT_SYNTH_MODEL

  const encoder = new TextEncoder()
  const stream = new ReadableStream<Uint8Array>({
    async start(controller) {
      const send = (event: DeepResearchEvent) => {
        controller.enqueue(encoder.encode(`data: ${JSON.stringify(event)}\n\n`))
      }

      try {
        const planRaw = await callOpenRouterJSON(
          apiKey,
          fastModel,
          "You are Kurai's learning-path planner. You only output valid JSON.",
          planPrompt(query),
          PLAN_MAX_TOKENS
        )
        const planObj = (planRaw ?? {}) as Record<string, unknown>
        const topicTitle = typeof planObj.title === 'string' && planObj.title.trim() ? planObj.title.trim() : query
        const overview = typeof planObj.overview === 'string' ? planObj.overview.trim() : ''
        const plannedSteps: PlannedStep[] = (Array.isArray(planObj.steps) ? planObj.steps : [])
          .map((s) => {
            const o = (s ?? {}) as Record<string, unknown>
            return { title: typeof o.title === 'string' ? o.title.trim() : '', summary: typeof o.summary === 'string' ? o.summary.trim() : '' }
          })
          .filter((s) => s.title.length > 0)
          .slice(0, MAX_STEPS)

        if (plannedSteps.length === 0 || !overview) {
          send({ type: 'error', message: 'Could not plan a learning path for this query.' })
          return
        }
        send({ type: 'plan', title: topicTitle, overview, steps: plannedSteps })

        const perStep = await Promise.all(
          plannedSteps.map(async (st) => {
            const results = await fetchSearchResults(`${topicTitle} ${st.title}`, MAX_SOURCES_PER_STEP, cookie)
            send({ type: 'searching', step: st.title, count: results.length })
            return results.map((result): StepCandidate => ({ result, step: st.title }))
          })
        )
        const candidates: StepCandidate[] = perStep.flat()

        if (candidates.length === 0) {
          send({ type: 'error', message: 'No sources found for this topic.' })
          return
        }

        const summaryModel = floorVariant(fastModel)
        let doneCount = 0
        interface SourceOutcome {
          candidate: StepCandidate
          digest: SourceDigest
          facts: string[]
        }
        const outcomes: SourceOutcome[] = (
          await Promise.all(
            candidates.map(async (c): Promise<SourceOutcome | null> => {
              try {
                const digest = await fetchDigest(c.result.url, c.step, cookie)
                if (!digest) return null
                const raw = await callOpenRouterJSON(
                  apiKey,
                  summaryModel,
                  "You are Kurai's research assistant. You only output valid JSON.",
                  summarizePrompt(topicTitle, c.step, digest),
                  SUMMARY_MAX_TOKENS
                )
                const o = (raw ?? {}) as Record<string, unknown>
                const facts = (Array.isArray(o.facts) ? o.facts : []).filter(
                  (f): f is string => typeof f === 'string' && f.trim().length > 0
                )
                return { candidate: c, digest, facts }
              } catch {
                return null
              } finally {
                doneCount++
                send({ type: 'reading', done: doneCount, total: candidates.length })
              }
            })
          )
        ).filter((o): o is SourceOutcome => o !== null && o.facts.length > 0)

        if (outcomes.length === 0) {
          send({ type: 'error', message: 'Could not extract anything useful from the sources found.' })
          return
        }

        const stepFacts: StepFact[] = plannedSteps.map((st) => ({
          step: st.title,
          facts: outcomes.filter((o) => o.candidate.step === st.title).flatMap((o) => o.facts),
        }))

        send({ type: 'synthesizing' })
        const assembleRaw = await callOpenRouterJSON(
          apiKey,
          synthModel,
          'You are Kurai, a careful research assistant. You only output valid JSON.',
          assemblePrompt(topicTitle, overview, plannedSteps, stepFacts, language),
          ASSEMBLE_MAX_TOKENS
        )
        const assembleObj = (assembleRaw ?? {}) as Record<string, unknown>
        const assembledSteps = Array.isArray(assembleObj.steps) ? assembleObj.steps : []
        const followups = (Array.isArray(assembleObj.followups) ? assembleObj.followups : [])
          .filter((f): f is string => typeof f === 'string' && f.trim().length > 0)
          .slice(0, 3)

        const steps: DeepResearchStep[] = plannedSteps.map((st, i) => {
          const a = (assembledSteps[i] ?? {}) as Record<string, unknown>
          const desc = typeof a.desc === 'string' ? a.desc.trim() : ''
          const facts: DeepResearchFact[] = (Array.isArray(a.facts) ? a.facts : [])
            .filter((f): f is string => typeof f === 'string' && f.trim().length > 0)
            .slice(0, 5)
            .map((text, j) => ({ n: String(j + 1), text }))
          return {
            id: `step-${i}`,
            n: String(i + 1).padStart(2, '0'),
            title: st.title,
            summary: st.summary,
            time: estimateStepTime(desc, facts.map((f) => f.text)),
            isStart: i === 0,
            desc,
            facts,
          }
        })

        if (steps.every((s) => s.desc === '' && s.facts.length === 0)) {
          send({ type: 'error', message: 'Empty learning path.' })
          return
        }

        const sourceByUrl = new Map<string, SearchResult>()
        for (const o of outcomes) {
          if (sourceByUrl.has(o.candidate.result.url)) continue
          sourceByUrl.set(o.candidate.result.url, { ...o.candidate.result, title: o.digest.title || o.candidate.result.title })
        }
        const sources = Array.from(sourceByUrl.values())

        const totalFacts = steps.reduce((n, s) => n + s.facts.length, 0)
        send({
          type: 'done',
          result: {
            title: topicTitle,
            overview,
            metaTime: totalReadTime(steps),
            metaSteps: steps.length,
            metaFacts: totalFacts,
            steps,
            sources,
            followups,
          },
        })
      } catch (err) {
        console.error('Deep research error:', err)
        send({ type: 'error', message: 'Deep research failed unexpectedly.' })
      } finally {
        controller.close()
      }
    },
  })

  return new Response(stream, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache, no-transform',
      Connection: 'keep-alive',
    },
  })
}
