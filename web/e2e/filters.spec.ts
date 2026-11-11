import { test, expect, type Page } from '@playwright/test'
import { interceptEmptyBoard, submitFromHome, RESULTS, META } from './fixtures'

async function recordSearches(page: Page, meta: Record<string, unknown> = {}): Promise<URLSearchParams[]> {
  const seen: URLSearchParams[] = []
  await interceptEmptyBoard(page)
  await page.route('**/api/search**', async (route) => {
    const params = new URL(route.request().url()).searchParams
    seen.push(params)
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      json: { query: params.get('q') ?? '', results: RESULTS, meta: { ...META, ...meta } },
    })
  })
  return seen
}

const last = (seen: URLSearchParams[]) => seen[seen.length - 1]

test('an unfiltered search sends no filter parameters', async ({ page }) => {
  const seen = await recordSearches(page)
  await submitFromHome(page, 'plain query')
  await expect(page.getByText('Alpha result')).toBeVisible()
  expect(last(seen).has('include_domains')).toBe(false)
  expect(last(seen).has('date_from')).toBe(false)
})

test('the site field filters by domain and rides the URL', async ({ page }) => {
  const seen = await recordSearches(page)
  await submitFromHome(page, 'custody keys')
  await expect(page.getByText('Alpha result')).toBeVisible()

  await page.locator('#folio-filter-site').fill('https://www.NorthernTrust.com/insights, reuters.com')
  await page.locator('#folio-filter-site').press('Enter')

  await expect.poll(() => last(seen).get('include_domains')).toBe('northerntrust.com,reuters.com')
  expect(last(seen).get('q')).toBe('custody keys')
  await expect(page).toHaveURL(/site=northerntrust\.com%2Creuters\.com/)
  await expect(page.locator('#folio-filter-site')).toHaveValue('northerntrust.com, reuters.com')
})

test('a date range is sent and says how undated results are treated', async ({ page }) => {
  const seen = await recordSearches(page, { time_range: 'year' })
  await submitFromHome(page, 'press monitoring')
  await expect(page.getByText('Alpha result')).toBeVisible()

  await page.locator('#folio-filter-from').fill('2026-01-01')
  await expect.poll(() => last(seen).get('date_from')).toBe('2026-01-01')
  expect(last(seen).has('date_strict')).toBe(false)
  expect(last(seen).has('date_to')).toBe(false)
  await expect(page.locator('[data-date-notice]')).toContainText('the engines limit them to the past year')
  await expect(page).toHaveURL(/from=2026-01-01/)
})

test('site: typed in the box becomes the site filter', async ({ page }) => {
  const seen = await recordSearches(page)
  await submitFromHome(page, 'custody keys site:northerntrust.com')
  await expect(page.getByRole('heading', { level: 1, name: 'custody keys', exact: true })).toBeVisible()
  expect(last(seen).get('q')).toBe('custody keys')
  expect(last(seen).get('include_domains')).toBe('northerntrust.com')
  await expect(page.locator('#folio-filter-site')).toHaveValue('northerntrust.com')
  await expect(page.locator('#folio-search')).toHaveValue('custody keys')
})

test('a range the engines cannot narrow says the undated results are unfiltered', async ({ page }) => {
  await recordSearches(page)
  await page.goto('/?q=old+news&to=2020-12-31')
  await expect(page.locator('[data-date-notice]')).toContainText('they are shown as they came')
})

test('no date notice without a date filter', async ({ page }) => {
  await recordSearches(page)
  await page.goto('/?q=plain&site=example.com')
  await expect(page.getByText('Alpha result')).toBeVisible()
  await expect(page.locator('[data-date-notice]')).toHaveCount(0)
})

test('a filtered URL restores the query and the filter row', async ({ page }) => {
  const seen = await recordSearches(page)
  await page.goto('/?q=custody+keys&site=northerntrust.com&from=2026-01-01&to=2026-06-30')
  await expect(page.getByText('Alpha result')).toBeVisible()
  expect(last(seen).get('include_domains')).toBe('northerntrust.com')
  expect(last(seen).get('date_from')).toBe('2026-01-01')
  expect(last(seen).get('date_to')).toBe('2026-06-30')
  await expect(page.locator('#folio-filter-site')).toHaveValue('northerntrust.com')
  await expect(page.locator('#folio-filter-from')).toHaveValue('2026-01-01')
  await expect(page.locator('#folio-filter-to')).toHaveValue('2026-06-30')
})

test('filters persist across facets and are cleared in one step', async ({ page }) => {
  const seen = await recordSearches(page)
  await page.goto('/?q=custody+keys&site=northerntrust.com')
  await expect(page.getByText('Alpha result')).toBeVisible()

  await page.getByRole('button', { name: /^Guides/ }).click()
  await expect.poll(() => last(seen).get('content_type')).toBe('guide')
  expect(last(seen).get('include_domains')).toBe('northerntrust.com')

  await page.getByRole('button', { name: 'Clear filters' }).click()
  await expect.poll(() => last(seen).has('include_domains')).toBe(false)
  await expect(page).toHaveURL(/\/\?q=custody\+keys$/)
  await expect(page.getByRole('button', { name: 'Clear filters' })).toHaveCount(0)
})

test('a relaxed phrase says the exact phrase was not found', async ({ page }) => {
  await recordSearches(page, { phrase_relaxed: true })
  await submitFromHome(page, '"Northern Trust" HSM')
  await expect(page.getByRole('status').filter({ hasText: 'No result contains the exact quoted phrase' })).toBeVisible()
})

test('no phrase notice when the phrase was honoured', async ({ page }) => {
  await recordSearches(page, { phrase_filtered: 3 })
  await submitFromHome(page, '"Northern Trust" HSM')
  await expect(page.getByText('Alpha result')).toBeVisible()
  await expect(page.getByText('No result contains the exact quoted phrase')).toHaveCount(0)
})
