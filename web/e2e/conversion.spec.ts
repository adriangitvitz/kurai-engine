import { test, expect } from '@playwright/test'
import { interceptSearch, submitFromHome, trackAIRoutes, CURRENCY, CORRECTIONS } from './fixtures'

const CARD = { name: 'Currency conversion: EUR to MXN' }

test('displayed precision bounded: minor units + ≤6 significant digits, EN format', async ({ page }) => {
  await interceptSearch(page, { currency: CURRENCY })
  trackAIRoutes(page)
  await submitFromHome(page, '300 euros to mxn')

  const card = page.getByRole('region', CARD)
  await expect(card).toBeVisible()
  await expect(card).toContainText('5,984.12 MXN')
  await expect(card).toContainText('rate 19.9471')
  await expect(card).not.toContainText('5984.11736049')
})

test('card shows as-of date and source, machine-readable data-asof', async ({ page }) => {
  await interceptSearch(page, { currency: CURRENCY })
  trackAIRoutes(page)
  await submitFromHome(page, '300 euros to mxn')

  const card = page.getByRole('region', CARD)
  await expect(card).toContainText(/rate as of 2026-07-17/i)
  await expect(card).toContainText(/frankfurter/i)
  await expect(card.locator('[data-asof]')).toHaveAttribute('data-asof', '2026-07-17')
})

test('both resolved ISO codes visible on an alias-resolved conversion', async ({ page }) => {
  await interceptSearch(page, { currency: CURRENCY })
  trackAIRoutes(page)
  await submitFromHome(page, '300 euros to pesos')

  const card = page.getByRole('region', CARD)
  await expect(card).toContainText('EUR')
  await expect(card).toContainText('MXN')
})

test('card renders identically regardless of the answer-language toggle', async ({ page }) => {
  await interceptSearch(page, { currency: CURRENCY })
  trackAIRoutes(page)
  await submitFromHome(page, '100 usd to eur')

  const before = await page.getByRole('region', CARD).textContent()
  await page.getByRole('button', { name: 'ES', exact: true }).click()
  const after = await page.getByRole('region', CARD).textContent()
  expect(after).toBe(before)
})

test('default is on: fresh browser renders the card untouched', async ({ page }) => {
  await interceptSearch(page, { currency: CURRENCY })
  trackAIRoutes(page)
  await submitFromHome(page, '300 euros to mxn')
  await expect(page.getByRole('region', CARD)).toBeVisible()
})

test('disabling hides rendered cards immediately; re-enabling restores', async ({ page }) => {
  await interceptSearch(page, { currency: CURRENCY })
  trackAIRoutes(page)
  await submitFromHome(page, '300 euros to mxn')

  const card = page.getByRole('region', CARD)
  await expect(card).toBeVisible()
  const sourcesBefore = await page.getByRole('heading', { level: 3 }).allTextContents()

  const sw = page.getByRole('switch', { name: 'Answer cards & panels' })
  await sw.click()
  await expect(card).toBeHidden()
  expect(await page.getByRole('heading', { level: 3 }).allTextContents()).toEqual(sourcesBefore)

  await sw.click()
  await expect(card).toBeVisible()
})

test('preference persists across contexts via storage state', async ({ page, browser }) => {
  await interceptSearch(page, { currency: CURRENCY })
  trackAIRoutes(page)
  await submitFromHome(page, '300 euros to mxn')
  await page.getByRole('switch', { name: 'Answer cards & panels' }).click()
  await expect(page.getByRole('region', CARD)).toBeHidden()

  const state = await page.context().storageState()
  const ctx = await browser.newContext({ storageState: state })
  const page2 = await ctx.newPage()
  await interceptSearch(page2, { currency: CURRENCY })
  trackAIRoutes(page2)
  await page2.goto('/')
  const input = page2.locator('#folio-search')
  await input.fill('300 euros to mxn')
  await input.press('Enter')

  await expect(page2.getByRole('switch', { name: 'Answer cards & panels' })).toHaveAttribute('aria-checked', 'false')
  await expect(page2.getByRole('region', CARD)).toBeHidden()
  await ctx.close()
})

test('toggling preserves the reading position on a multi-entry thread', async ({ page }) => {
  await interceptSearch(page, { currency: CURRENCY, corrections: CORRECTIONS })
  trackAIRoutes(page)
  await submitFromHome(page, '300 euros to mxn')

  await page.getByRole('group', { name: 'Did you mean' }).first().getByRole('button', { name: CORRECTIONS[0] }).click()
  const secondHeading = page.getByRole('heading', { level: 1, name: /mechanical keyboards/i })
  await expect(secondHeading).toBeVisible()

  await secondHeading.scrollIntoViewIfNeeded()
  const before = await secondHeading.boundingBox()

  await page
    .getByRole('switch', { name: 'Answer cards & panels' })
    .evaluate((el) => (el as HTMLElement).click())
  await page.waitForTimeout(100)
  const after = await secondHeading.boundingBox()

  expect(Math.abs((after?.y ?? 0) - (before?.y ?? 0))).toBeLessThanOrEqual(24)
})

test('card renders in Sources-only mode without AI calls', async ({ page }) => {
  await interceptSearch(page, { currency: CURRENCY })
  const ai = trackAIRoutes(page)
  await submitFromHome(page, '300 euros to mxn')

  await expect(page.getByRole('region', CARD)).toBeVisible()
  expect(ai.synthesize).toBe(0)
  expect(ai.deepResearch).toBe(0)
})

for (const mode of ['sources', 'synth'] as const) {
  test(`provenance label visible in ${mode} mode`, async ({ page }) => {
    await interceptSearch(page, { currency: CURRENCY })
    trackAIRoutes(page)

    await page.goto('/')
    if (mode === 'synth') await page.getByRole('button', { name: 'Synthesized answer' }).click()
    const input = page.locator('#folio-search')
    await input.fill('300 euros to mxn')
    await input.press('Enter')

    const card = page.getByRole('region', CARD)
    await expect(card).toBeVisible()
    await expect(card).toContainText(/conversion/i)
    await expect(card).toContainText(/frankfurter/i)
  })
}

test('card carries no reserved AI markers; AnswerZone keeps its AI label', async ({ page }) => {
  await interceptSearch(page, { currency: CURRENCY })
  trackAIRoutes(page)

  await page.goto('/')
  await page.getByRole('button', { name: 'Synthesized answer' }).click()
  const input = page.locator('#folio-search')
  await input.fill('300 euros to mxn')
  await input.press('Enter')

  const card = page.getByRole('region', CARD)
  await expect(card).toBeVisible()
  await expect(page.getByText('Stub synthesis.')).toBeVisible()

  expect(await card.locator('svg path[d^="M12 2l1.8"]').count()).toBe(0)
  const amberButtons = await card.locator('button').evaluateAll((els) =>
    els.filter((el) => getComputedStyle(el).backgroundColor === 'rgb(245, 194, 87)').length
  )
  expect(amberButtons).toBe(0)
})

test('conversion card exposed as a labeled region with class + subject', async ({ page }) => {
  await interceptSearch(page, { currency: CURRENCY })
  trackAIRoutes(page)
  await submitFromHome(page, '300 euros to mxn')
  await expect(page.getByRole('region', { name: 'Currency conversion: EUR to MXN' })).toBeVisible()
})

test('conversion card reflows at 320px with units and ISO codes intact', async ({ browser }) => {
  const ctx = await browser.newContext({ viewport: { width: 320, height: 800 } })
  const p = await ctx.newPage()
  await interceptSearch(p, { currency: CURRENCY })
  trackAIRoutes(p)
  await p.goto('/')
  const input = p.locator('#folio-search')
  await input.fill('300 euros to mxn')
  await input.press('Enter')

  const card = p.getByRole('region', CARD)
  await expect(card).toBeVisible()
  await expect(card).toContainText('5,984.12 MXN')
  await expect(card).toContainText('EUR')

  const overflow = await p.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth
  )
  expect(overflow).toBeLessThanOrEqual(0)
  await ctx.close()
})
