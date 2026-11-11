import type { Page } from '@playwright/test'
import type { CurrencyAnswer, WeatherAnswer, InfoboxAnswer } from '../lib/types'

export const RESULTS = [
  {
    url: 'https://alpha.example/a',
    title: 'Alpha result',
    snippet: 'first snippet',
    engines: ['bing'],
    original_score: 10,
    final_score: 0.9,
    content_type: 'article',
  },
  {
    url: 'https://beta.example/b',
    title: 'Beta result',
    snippet: 'second snippet',
    engines: ['yahoo'],
    original_score: 5,
    final_score: 0.5,
    content_type: 'guide',
  },
]

export const META = {
  total: RESULTS.length,
  page: 1,
  page_size: 10,
  total_pages: 1,
  blocklist_filtered: 0,
  processing_time_ms: 12,
  from_cache: false,
  facets: { article: 1, study: 0, video: 0, discussion: 0, guide: 1 },
}

export const CORRECTIONS = ['mechanical keyboards', 'mechanical keyboard switches']

export const CURRENCY = {
  base: 'EUR',
  quote: 'MXN',
  amount: 300,
  converted: 5984.11736049,
  rate: 19.9470578683,
  rate_date: '2026-07-17',
  source: 'frankfurter',
  minor_units: 2,
}

export const WEATHER = {
  place: 'Oaxaca City, Mexico',
  country_code: 'MX',
  timezone: 'America/Mexico_City',
  temperature_c: 24.1,
  condition: 'Partly cloudy',
  forecast: [
    { date: '2026-07-17', min_c: 15.2, max_c: 27.8, condition: 'Partly cloudy' },
    { date: '2026-07-18', min_c: 15.0, max_c: 28.1, condition: 'Sunny' },
  ],
  source: 'open-meteo',
  as_of: '2026-07-17T14:00:00-06:00',
}

export const INFOBOX = {
  title: 'Albert Einstein',
  summary: 'German-born theoretical physicist who developed the theory of relativity.',
  attributes: [{ label: 'Born', value: '14 March 1879' }],
  links: [{ title: 'Wikipedia', url: 'https://en.wikipedia.org/wiki/Albert_Einstein' }],
  source: 'wikipedia',
}

export interface AnswersFixture {
  currency?: CurrencyAnswer
  weather?: WeatherAnswer
  infobox?: InfoboxAnswer
  corrections?: string[]
}

export async function interceptEmptyBoard(page: Page) {
  await page.route('**/api/bookmarks**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', json: { bookmarks: [] } })
      return
    }
    await route.fulfill({ status: 200, contentType: 'application/json', json: {} })
  })
}

export async function interceptSearch(page: Page, answers?: AnswersFixture) {
  await interceptEmptyBoard(page)
  await page.route('**/api/search**', async (route) => {
    const url = new URL(route.request().url())
    const body: Record<string, unknown> = {
      query: url.searchParams.get('q') ?? '',
      results: RESULTS,
      meta: META,
    }
    if (answers && Object.keys(answers).length > 0) body.answers = answers
    await route.fulfill({ status: 200, contentType: 'application/json', json: body })
  })
}

export function trackAIRoutes(page: Page): { synthesize: number; deepResearch: number } {
  const counts = { synthesize: 0, deepResearch: 0 }
  void page.route('**/api/synthesize**', async (route) => {
    counts.synthesize++
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      json: { paragraphs: ['Stub synthesis.'], facts: [], agreement: '', followups: [] },
    })
  })
  void page.route('**/api/deep-research**', async (route) => {
    counts.deepResearch++
    await route.fulfill({ status: 503, contentType: 'application/json', json: { error: 'stubbed off' } })
  })
  return counts
}

export async function submitFromHome(page: Page, query: string) {
  await page.goto('/')
  const input = page.locator('#folio-search')
  await input.fill(query)
  await input.press('Enter')
}
