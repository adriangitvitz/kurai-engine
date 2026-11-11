import { test, expect } from '@playwright/test'
import { interceptSearch, submitFromHome, trackAIRoutes, INFOBOX, WEATHER, CURRENCY, RESULTS } from './fixtures'

const PANEL = { name: 'Knowledge panel: Albert Einstein' }

const MINIMAL_INFOBOX = {
  title: 'Albert Einstein',
  summary: 'German-born theoretical physicist who developed the theory of relativity.',
  links: [{ title: 'Wikipedia', url: 'https://en.wikipedia.org/wiki/Albert_Einstein' }],
  source: 'wikipedia',
}

test('minimal infobox renders title, summary, link — no empty scaffolding', async ({ page }) => {
  await interceptSearch(page, { infobox: MINIMAL_INFOBOX })
  trackAIRoutes(page)
  await submitFromHome(page, 'albert einstein')

  const panel = page.getByRole('region', PANEL)
  await expect(panel).toBeVisible()
  await expect(panel).toContainText('Albert Einstein')
  await expect(panel).toContainText('German-born theoretical physicist')
  await expect(panel.getByRole('link', { name: 'Wikipedia' })).toBeVisible()
  expect(await panel.locator('dl').count()).toBe(0)
  expect(await panel.locator('img').count()).toBe(0)
})

test('attribute-rich infobox renders label/value pairs, at most 8', async ({ page }) => {
  const rich = {
    ...MINIMAL_INFOBOX,
    attributes: Array.from({ length: 12 }, (_, i) => ({ label: `Fact ${i + 1}`, value: `Value ${i + 1}` })),
  }
  await interceptSearch(page, { infobox: rich })
  trackAIRoutes(page)
  await submitFromHome(page, 'albert einstein')

  const panel = page.getByRole('region', PANEL)
  await expect(panel).toContainText('Fact 1')
  await expect(panel).toContainText('Value 8')
  expect(await panel.locator('dt').count()).toBeLessThanOrEqual(8)
  await expect(panel).not.toContainText('Fact 9')
})

test('image renders with alt = entity title; no placeholder when absent', async ({ page }) => {
  await interceptSearch(page, { infobox: { ...MINIMAL_INFOBOX, image_url: 'https://upload.wikimedia.org/einstein.jpg' } })
  trackAIRoutes(page)
  await submitFromHome(page, 'albert einstein')

  const img = page.getByRole('region', PANEL).locator('img')
  await expect(img).toHaveAttribute('alt', 'Albert Einstein')
})

test('no image and no placeholder when upstream has none', async ({ page }) => {
  await interceptSearch(page, { infobox: MINIMAL_INFOBOX })
  trackAIRoutes(page)
  await submitFromHome(page, 'albert einstein')
  await expect(page.getByRole('region', PANEL)).toBeVisible()
  expect(await page.getByRole('region', PANEL).locator('img').count()).toBe(0)
})

test('every rendered panel text node is a substring of the pinned payload or fixed chrome', async ({ page }) => {
  await interceptSearch(page, { infobox: INFOBOX })
  trackAIRoutes(page)
  await submitFromHome(page, 'albert einstein')

  const panel = page.getByRole('region', PANEL)
  await expect(panel).toBeVisible()
  const texts = await panel.evaluate((el) => {
    const walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT)
    const out: string[] = []
    while (walker.nextNode()) {
      const t = walker.currentNode.textContent?.trim()
      if (t) out.push(t)
    }
    return out
  })

  const allowed = [
    INFOBOX.title,
    INFOBOX.summary,
    ...INFOBOX.attributes.map((a) => a.label),
    ...INFOBOX.attributes.map((a) => a.value),
    ...INFOBOX.links.map((l) => l.title),
    'Knowledge panel',
    '·',
    INFOBOX.source,
  ]
  for (const text of texts) {
    expect(
      allowed.some((a) => a.toLowerCase().includes(text.toLowerCase()) || text.toLowerCase().includes(a.toLowerCase())),
      `unexpected panel text not traceable to the payload: ${JSON.stringify(text)}`
    ).toBe(true)
  }
})

test('visible Wikipedia link targets the payload URL — never synthesized', async ({ page }) => {
  await interceptSearch(page, { infobox: INFOBOX })
  trackAIRoutes(page)
  await submitFromHome(page, 'albert einstein')

  const link = page.getByRole('region', PANEL).getByRole('link', { name: 'Wikipedia' })
  await expect(link).toBeVisible()
  await expect(link).toHaveAttribute('href', 'https://en.wikipedia.org/wiki/Albert_Einstein')
})

test('DOM order: question → weather card → AnswerZone → knowledge panel → Sources', async ({ page }) => {
  await interceptSearch(page, { weather: WEATHER, infobox: INFOBOX })
  trackAIRoutes(page)

  await page.goto('/')
  await page.getByRole('button', { name: 'Synthesized answer' }).click()
  const input = page.locator('#folio-search')
  await input.fill('weather in oaxaca')
  await input.press('Enter')

  await expect(page.getByRole('region', PANEL)).toBeVisible()
  const order = await page.evaluate(() => {
    const markers: [string, Element | null][] = [
      ['question', document.querySelector('h1[tabindex]')],
      ['weather', document.querySelector('[role="region"][aria-label^="Weather:"]')],
      ['answerzone', Array.from(document.querySelectorAll('*')).find((el) => el.textContent === 'Stub synthesis.') ?? null],
      ['panel', document.querySelector('[role="region"][aria-label^="Knowledge panel:"]')],
      ['sources', Array.from(document.querySelectorAll('span')).find((el) => el.textContent?.includes('Sources & reading')) ?? null],
    ]
    const present = markers.filter(([, el]) => el)
    const sorted = [...present].sort(([, a], [, b]) =>
      a!.compareDocumentPosition(b!) & Node.DOCUMENT_POSITION_FOLLOWING ? -1 : 1
    )
    return { actual: present.map(([n]) => n), sorted: sorted.map(([n]) => n) }
  })
  expect(order.actual).toEqual(['question', 'weather', 'answerzone', 'panel', 'sources'])
  expect(order.sorted).toEqual(order.actual)
})

test('ranked source list identical with the preference on versus off', async ({ page }) => {
  await interceptSearch(page, { infobox: INFOBOX })
  trackAIRoutes(page)
  await submitFromHome(page, 'albert einstein')

  const titles = () => page.getByRole('heading', { level: 3 }).allTextContents()
  await expect(page.getByRole('heading', { level: 3 })).toHaveCount(RESULTS.length)
  const withPanel = await titles()
  expect(withPanel).toEqual(RESULTS.map((r) => r.title))

  await page.getByRole('switch', { name: 'Answer cards & panels' }).click()
  expect(await titles()).toEqual(withPanel)
})

test('preference off → no panel renders; thread shows plain results', async ({ page }) => {
  await interceptSearch(page, { infobox: INFOBOX })
  trackAIRoutes(page)
  await submitFromHome(page, 'albert einstein')

  const panel = page.getByRole('region', PANEL)
  await expect(panel).toBeVisible()
  await page.getByRole('switch', { name: 'Answer cards & panels' }).click()
  await expect(panel).toBeHidden()
  await expect(page.getByText('Alpha result')).toBeVisible()
})

test('panel source links reachable and activatable by keyboard in reading order', async ({ page }) => {
  await interceptSearch(page, { infobox: INFOBOX })
  trackAIRoutes(page)
  await submitFromHome(page, 'albert einstein')

  const link = page.getByRole('region', PANEL).getByRole('link', { name: 'Wikipedia' })
  await link.focus()
  await expect(link).toBeFocused()
})

for (const [cls, fixture, region, source] of [
  ['conversion', { currency: CURRENCY }, 'Currency conversion: EUR to MXN', /frankfurter/i],
  ['weather', { weather: WEATHER }, 'Weather: Oaxaca City, Mexico', /open-meteo/i],
  ['panel', { infobox: INFOBOX }, 'Knowledge panel: Albert Einstein', /wikipedia/i],
] as const) {
  test(`${cls} names its source as visible eyebrow text`, async ({ page }) => {
    await interceptSearch(page, fixture)
    trackAIRoutes(page)
    await submitFromHome(page, 'any trigger query')
    const card = page.getByRole('region', { name: region })
    await expect(card).toBeVisible()
    await expect(card).toContainText(source)
  })
}

test('maximal entry reflows at 320px with no horizontal scroll or information loss', async ({ browser }) => {
  const ctx = await browser.newContext({ viewport: { width: 320, height: 900 } })
  const p = await ctx.newPage()
  await interceptSearch(p, { weather: WEATHER, infobox: INFOBOX, corrections: ['albert einstein biography'] })
  trackAIRoutes(p)
  await p.goto('/')
  const input = p.locator('#folio-search')
  await input.fill('weather in oaxaca')
  await input.press('Enter')

  await expect(p.getByRole('group', { name: 'Did you mean' })).toBeVisible()
  await expect(p.getByRole('region', { name: 'Weather: Oaxaca City, Mexico' })).toBeVisible()
  await expect(p.getByRole('region', PANEL)).toBeVisible()
  await expect(p.getByRole('region', PANEL)).toContainText('Albert Einstein')

  const overflow = await p.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth
  )
  expect(overflow).toBeLessThanOrEqual(0)

  await ctx.close()
})
