import { test, expect, type Page } from '@playwright/test'
import { interceptSearch, submitFromHome, trackAIRoutes, RESULTS, META } from './fixtures'

const heading = (page: Page, name: string) => page.getByRole('heading', { level: 1, name, exact: true })

test('the search box keeps one id on home and on results', async ({ page }) => {
  await interceptSearch(page)
  await page.goto('/')
  await expect(page.locator('form[role="search"] #folio-search[name="q"]')).toHaveCount(1)

  await submitFromHome(page, 'stable box')
  await expect(heading(page, 'stable box')).toBeVisible()
  await expect(page.locator('form[role="search"] #folio-search[name="q"]')).toHaveCount(1)
  await expect(page.locator('#folio-search')).toHaveValue('stable box')
})

test('slash focuses the search box', async ({ page }) => {
  await interceptSearch(page)
  await submitFromHome(page, 'slash focus')
  await expect(heading(page, 'slash focus')).toBeVisible()
  await page.locator('#folio-search').blur()
  await page.keyboard.press('/')
  await expect(page.locator('#folio-search')).toBeFocused()
  await expect(page.locator('#folio-search')).toHaveValue('slash focus')
})

test('submitting a query writes it to the URL', async ({ page }) => {
  await interceptSearch(page)
  await submitFromHome(page, 'url state')
  await expect(heading(page, 'url state')).toBeVisible()
  await expect(page).toHaveURL(/\/\?q=url\+state$/)
})

test('opening /?q= searches without visiting home', async ({ page }) => {
  await interceptSearch(page)
  await page.goto('/?q=direct%20link')
  await expect(heading(page, 'direct link')).toBeVisible()
  await expect(page.getByText('Alpha result')).toBeVisible()
})

test('the mode rides the URL', async ({ page }) => {
  await interceptSearch(page)
  const ai = trackAIRoutes(page)
  await page.goto('/?q=synth%20link&mode=synth')
  await expect(page.getByText('Stub synthesis.')).toBeVisible()
  expect(ai.synthesize).toBe(1)
})

test('back and forward replay the query of each URL', async ({ page }) => {
  await interceptSearch(page)
  await submitFromHome(page, 'first query')
  await expect(heading(page, 'first query')).toBeVisible()

  await page.locator('#folio-search').fill('second query')
  await page.locator('#folio-search').press('Enter')
  await expect(heading(page, 'second query')).toBeVisible()
  await expect(page).toHaveURL(/q=second\+query/)

  await page.goBack()
  await expect(heading(page, 'first query')).toBeVisible()
  await expect(heading(page, 'second query')).toHaveCount(0)

  await page.goBack()
  await expect(page.getByPlaceholder('Ask anything, or paste a link')).toBeVisible()

  await page.goForward()
  await expect(heading(page, 'first query')).toBeVisible()
})

test('a reload keeps the search', async ({ page }) => {
  await interceptSearch(page)
  await submitFromHome(page, 'survives reload')
  await expect(heading(page, 'survives reload')).toBeVisible()
  await page.reload()
  await expect(heading(page, 'survives reload')).toBeVisible()
})

test('the logo returns to / and clears the query from the URL', async ({ page }) => {
  await interceptSearch(page)
  await submitFromHome(page, 'going home')
  await expect(heading(page, 'going home')).toBeVisible()
  await page.getByRole('button', { name: 'Kurai home' }).click()
  await expect(page).toHaveURL(/\/$/)
})

test('the heading shows the query exactly as typed', async ({ page }) => {
  await interceptSearch(page)
  await submitFromHome(page, 'not a question at all')
  await expect(heading(page, 'not a question at all')).toBeVisible()
})

test('each card exposes its URL as text and as a real link', async ({ page }) => {
  await interceptSearch(page)
  await submitFromHome(page, 'card links')
  for (const r of RESULTS) {
    await expect(page.getByRole('link', { name: r.title, exact: true })).toHaveAttribute('href', r.url)
    await expect(page.locator('[data-source-url]', { hasText: r.url })).toBeVisible()
  }
})

test('pagination names the visible range out of the total', async ({ page }) => {
  await interceptSearch(page)
  const many = Array.from({ length: 10 }, (_, i) => ({ ...RESULTS[0], url: `https://alpha.example/${i}`, title: `Result ${i}` }))
  await page.route('**/api/search**', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', json: { query: 'x', results: many, meta: { ...META, total: 10 } } })
  )
  await submitFromHome(page, 'ranges')
  await expect(page.getByText('1–6 of 10')).toBeVisible()
  await page.getByRole('button', { name: 'Next results' }).click()
  await expect(page.getByText('7–10 of 10')).toBeVisible()
})

test('a closed board contributes no visible or focusable content', async ({ page }) => {
  await interceptSearch(page)
  await page.route('**/api/bookmarks**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      json: { bookmarks: [{ url: 'https://saved.example/x', title: 'Unrelated saved item', saved_at: '2026-11-01T00:00:00Z' }] },
    })
  )
  await submitFromHome(page, 'closed board')
  await expect(heading(page, 'closed board')).toBeVisible()
  await expect(page.getByText('Unrelated saved item')).toHaveCount(0)

  await page.getByRole('button', { name: /^Board/ }).click()
  await expect(page.getByText('Unrelated saved item')).toBeVisible()
  await page.getByRole('button', { name: 'Close board' }).click()
  await expect(page.getByText('Unrelated saved item')).toHaveCount(0)

  await page.getByRole('button', { name: /^Board/ }).click()
  await expect(page.getByText('Unrelated saved item')).toBeVisible()
})

test('the result count is singular for one result', async ({ page }) => {
  await interceptSearch(page)
  await page.route('**/api/search**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      json: { query: 'x', results: [RESULTS[0]], meta: { ...META, total: 1, facets: { article: 1 } } },
    })
  )
  await submitFromHome(page, 'just one')
  await expect(page.getByText('1 result', { exact: true })).toBeVisible()
  await expect(page.getByText('1 results')).toHaveCount(0)
})
