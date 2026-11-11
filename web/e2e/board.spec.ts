import { test, expect, type Page } from '@playwright/test'
import { interceptSearch, submitFromHome, trackAIRoutes, RESULTS } from './fixtures'

interface FakeBoard {
  rows: Map<string, Record<string, unknown>>
  saves: number
  deletes: number
  failSave: boolean
}

async function interceptBookmarks(page: Page, opts: { failSave?: boolean } = {}): Promise<FakeBoard> {
  const board: FakeBoard = { rows: new Map(), saves: 0, deletes: 0, failSave: !!opts.failSave }

  await page.route('**/api/bookmarks**', async (route) => {
    const req = route.request()
    const url = new URL(req.url())

    if (req.method() === 'GET') {
      const bookmarks = [...board.rows.values()].reverse()
      await route.fulfill({ status: 200, contentType: 'application/json', json: { bookmarks } })
      return
    }

    if (req.method() === 'POST') {
      board.saves++
      if (board.failSave) {
        await route.fulfill({ status: 500, contentType: 'application/json', json: { error: 'boom' } })
        return
      }
      const body = req.postDataJSON() as Record<string, unknown>
      const key = String(body.url)
      const existing = board.rows.get(key)
      const stored = existing
        ? { ...existing, title: body.title, query: existing.query ?? body.query }
        : { ...body, saved_at: new Date().toISOString() }
      board.rows.set(key, stored)
      await route.fulfill({ status: 200, contentType: 'application/json', json: stored })
      return
    }

    if (req.method() === 'DELETE') {
      board.deletes++
      board.rows.delete(url.searchParams.get('url') ?? '')
      await route.fulfill({ status: 204, body: '' })
      return
    }
    await route.continue()
  })

  return board
}

const openBoard = (page: Page) => page.getByRole('button', { name: /^Board/ }).click()

test('a saved source persists across a full page reload', async ({ page }) => {
  await interceptSearch(page)
  const board = await interceptBookmarks(page)
  trackAIRoutes(page)

  await submitFromHome(page, 'mechanical keyboards')
  await page.getByRole('button', { name: `Save ${RESULTS[0].title} to board` }).click()
  await expect.poll(() => board.saves).toBe(1)

  await page.reload()
  await submitFromHome(page, 'anything at all')
  await openBoard(page)
  await expect(
    page.getByRole('link', { name: new RegExp(RESULTS[0].title) }).filter({ hasText: 'alpha.example' })
  ).toBeVisible()
})

test('deleting a bookmark removes it and it stays gone after reload', async ({ page }) => {
  await interceptSearch(page)
  const board = await interceptBookmarks(page)
  trackAIRoutes(page)

  await submitFromHome(page, 'mechanical keyboards')
  await page.getByRole('button', { name: `Save ${RESULTS[0].title} to board` }).click()
  await expect.poll(() => board.rows.size).toBe(1)

  await openBoard(page)
  await page.getByRole('button', { name: `Remove ${RESULTS[0].title}`, exact: true }).click()
  await expect.poll(() => board.deletes).toBe(1)
  await expect.poll(() => board.rows.size).toBe(0)

  await page.reload()
  await submitFromHome(page, 'anything at all')
  await openBoard(page)
  await expect(page.getByText('Tap the bookmark on any source to collect it here.')).toBeVisible()
})

test('a failed save rolls back — the board never asserts a save that did not happen', async ({ page }) => {
  await interceptSearch(page)
  const board = await interceptBookmarks(page, { failSave: true })
  trackAIRoutes(page)

  await submitFromHome(page, 'mechanical keyboards')
  await page.getByRole('button', { name: `Save ${RESULTS[0].title} to board` }).click()
  await expect.poll(() => board.saves).toBe(1)

  await openBoard(page)
  await expect(page.getByText('Tap the bookmark on any source to collect it here.')).toBeVisible()
})

const LEGACY = {
  'https://legacy.example/1': {
    id: 'https://legacy.example/1',
    url: 'https://legacy.example/1',
    title: 'Legacy bookmark',
    snippet: 'saved before the server existed',
    type: 'Article',
    rawType: 'article',
    domain: 'legacy.example',
    meta: '1 engine',
    score: 0.5,
    size: 'compact',
    hasImg: false,
    imgLabel: 'article',
  },
}

async function seedLegacy(page: Page) {
  await page.addInitScript((legacy) => {
    localStorage.setItem('folio.saved.v1', JSON.stringify(legacy))
  }, LEGACY)
}

test('an existing browser board imports once and does not duplicate or re-run', async ({ page }) => {
  await interceptSearch(page)
  const board = await interceptBookmarks(page)
  trackAIRoutes(page)
  await seedLegacy(page)

  await page.goto('/')
  await expect.poll(() => board.rows.size).toBe(1)
  expect(board.saves).toBe(1)

  const flag = await page.evaluate(() => localStorage.getItem('folio.saved.imported.v1'))
  expect(flag, 'the import flag is set so it never runs again').toBeTruthy()

  await page.reload()
  await page.waitForTimeout(300)
  expect(board.saves, 'no second import').toBe(1)
  expect(board.rows.size, 'no duplicate').toBe(1)
})

test('a failed import preserves the local data and does not set the flag', async ({ page }) => {
  await interceptSearch(page)
  await interceptBookmarks(page, { failSave: true })
  trackAIRoutes(page)
  await seedLegacy(page)

  await page.goto('/')
  await page.waitForTimeout(400)

  const state = await page.evaluate(() => ({
    legacy: localStorage.getItem('folio.saved.v1'),
    flag: localStorage.getItem('folio.saved.imported.v1'),
  }))
  expect(state.legacy, 'local data survives a failed import').toContain('Legacy bookmark')
  expect(state.flag, 'flag unset so the import retries on a later load').toBeNull()
})

test('the board shows the search that produced a bookmark, and when', async ({ page }) => {
  await interceptSearch(page)
  await interceptBookmarks(page)
  trackAIRoutes(page)

  await submitFromHome(page, 'clima en guadalajara')
  await page.getByRole('button', { name: `Save ${RESULTS[0].title} to board` }).click()
  await openBoard(page)

  await expect(page.getByText(/saved while searching “clima en guadalajara”/i)).toBeVisible()
  await expect(page.getByText(/saved \w{3} \d+/i)).toBeVisible()
})

test('the board groups bookmarks by search and returns to the flat list', async ({ page }) => {
  await interceptSearch(page)
  await interceptBookmarks(page)
  trackAIRoutes(page)

  await submitFromHome(page, 'clima en guadalajara')
  await page.getByRole('button', { name: `Save ${RESULTS[0].title} to board` }).click()
  await expect(page.getByRole('button', { name: `Remove ${RESULTS[0].title} from board` })).toBeVisible()

  const input = page.locator('#folio-search')
  await input.fill('mechanical keyboards')
  await input.press('Enter')
  await expect(page.getByRole('heading', { level: 1, name: /mechanical keyboards/i })).toBeVisible()
  await page.getByRole('button', { name: `Save ${RESULTS[1].title} to board` }).click()

  await openBoard(page)
  const boardView = page.getByRole('group', { name: 'Board view' })
  await boardView.getByRole('button', { name: 'By search' }).click()

  await expect(page.getByRole('region', { name: /saved while searching clima en guadalajara/i })).toBeVisible()
  await expect(page.getByRole('region', { name: /saved while searching mechanical keyboards/i })).toBeVisible()

  await boardView.getByRole('button', { name: 'All' }).click()
  await expect(page.getByText(/saved while searching/i).first()).toBeVisible()
})

test('a bookmark with no recorded search renders cleanly in both views', async ({ page }) => {
  await interceptSearch(page)
  await interceptBookmarks(page)
  trackAIRoutes(page)
  await seedLegacy(page)

  await page.goto('/')
  await submitFromHome(page, 'anything')
  await openBoard(page)

  await expect(page.getByText('Legacy bookmark')).toBeVisible()
  await expect(page.getByText(/saved while searching/i)).toHaveCount(0)

  const boardView = page.getByRole('group', { name: 'Board view' })
  await boardView.getByRole('button', { name: 'By search' }).click()
  await expect(page.getByText(/no search recorded · 1/i)).toBeVisible()
})
