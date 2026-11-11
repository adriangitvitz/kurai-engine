import { test, expect, type Page } from '@playwright/test'
import { interceptSearch, submitFromHome, RESULTS } from './fixtures'

const PAGE_URL = 'https://docs.example/guide/intro'

const ARTICLE = {
  url: PAGE_URL,
  final_url: PAGE_URL,
  title: 'An introduction to the guide',
  site_name: 'Example Docs',
  content_markdown: '# An introduction to the guide\n\n## First section\n\nBody paragraph with a [link](https://other.example/x).\n\n- one\n- two\n',
  word_count: 9,
}

interface ExtractFake {
  full: string[]
}

async function interceptExtract(page: Page, failure?: { status: number; code: string }): Promise<ExtractFake> {
  const fake: ExtractFake = { full: [] }
  await page.route('**/api/extract**', async (route) => {
    const params = new URL(route.request().url()).searchParams
    if (params.get('passages_only') === 'true') {
      await route.fulfill({ status: 200, contentType: 'application/json', json: { passages: [] } })
      return
    }
    fake.full.push(params.get('url') ?? '')
    if (failure) {
      await route.fulfill({
        status: failure.status,
        contentType: 'application/json',
        json: { error: { code: failure.code, message: 'upstream detail' } },
      })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      json: { result: { ...ARTICLE, url: params.get('url') } },
    })
  })
  return fake
}

function countSearches(page: Page): { n: number } {
  const count = { n: 0 }
  page.on('request', (req) => {
    if (new URL(req.url()).pathname === '/api/search') count.n++
  })
  return count
}

test('a pasted link is read, not searched', async ({ page }) => {
  await interceptSearch(page)
  const extract = await interceptExtract(page)
  const searches = countSearches(page)

  await submitFromHome(page, PAGE_URL)

  await expect(page.getByRole('heading', { level: 1, name: ARTICLE.title })).toBeVisible()
  await expect(page.locator('[data-page-url]')).toHaveAttribute('href', PAGE_URL)
  await expect(page.locator('[data-page-url]')).toContainText(PAGE_URL)
  await expect(page.getByRole('heading', { level: 2, name: 'First section' })).toBeVisible()
  await expect(page.getByRole('heading', { level: 1 })).toHaveCount(1)
  await expect(page.getByRole('link', { name: 'link', exact: true })).toHaveAttribute('href', 'https://other.example/x')
  await expect(page.getByText('Example Docs · 9 words')).toBeVisible()
  expect(extract.full).toEqual([PAGE_URL])
  expect(searches.n).toBe(0)
  await expect(page).toHaveURL(`/?url=${encodeURIComponent(PAGE_URL)}`)
})

test('a www link without a scheme is read as https', async ({ page }) => {
  await interceptSearch(page)
  const extract = await interceptExtract(page)
  await submitFromHome(page, 'www.example.org/a')
  await expect(page.getByRole('heading', { level: 1, name: ARTICLE.title })).toBeVisible()
  expect(extract.full).toEqual(['https://www.example.org/a'])
})

test('a dotted query that is not a link is still searched', async ({ page }) => {
  await interceptSearch(page)
  const extract = await interceptExtract(page)
  await submitFromHome(page, 'node.js')
  await expect(page.getByRole('heading', { level: 1, name: 'node.js', exact: true })).toBeVisible()
  await expect(page.getByText('Alpha result')).toBeVisible()
  expect(extract.full).toEqual([])
})

test('/?url= opens the reader directly', async ({ page }) => {
  await interceptSearch(page)
  await interceptExtract(page)
  await page.goto(`/?url=${encodeURIComponent(PAGE_URL)}`)
  await expect(page.getByRole('heading', { level: 1, name: ARTICLE.title })).toBeVisible()
  await expect(page.locator('#folio-search')).toHaveValue(PAGE_URL)
})

for (const [code, text] of [
  ['extraction_failed', 'This page has no readable article content.'],
  ['unsupported_content_type', 'This kind of file can’t be read here.'],
  ['fetch_timeout', 'The page took too long to respond. Try again in a moment.'],
] as const) {
  test(`a ${code} failure reads as a plain message`, async ({ page }) => {
    await interceptSearch(page)
    await interceptExtract(page, { status: 502, code })
    await submitFromHome(page, PAGE_URL)
    await expect(page.getByRole('alert').filter({ hasText: text })).toBeVisible()
    await expect(page.locator('[data-page-url]')).toHaveAttribute('href', PAGE_URL)
    await expect(page.getByText('upstream detail')).toHaveCount(0)
  })
}

test('Read here opens the page in the thread, below the results', async ({ page }) => {
  await interceptSearch(page)
  const extract = await interceptExtract(page)
  await submitFromHome(page, 'read here')

  const card = page.getByRole('link', { name: RESULTS[0].title, exact: true })
  await card.hover()
  await page.getByRole('button', { name: 'Read here' }).first().click()

  const heading = page.getByRole('heading', { level: 1, name: ARTICLE.title })
  await expect(heading).toBeVisible()
  await expect(heading).toBeFocused()
  await expect(page.getByRole('heading', { level: 1, name: 'read here', exact: true })).toBeVisible()
  expect(extract.full).toEqual([RESULTS[0].url])

  await page.goBack()
  await expect(page.getByRole('heading', { level: 1, name: ARTICLE.title })).toHaveCount(0)
  await expect(page.getByRole('heading', { level: 1, name: 'read here', exact: true })).toBeVisible()
})
