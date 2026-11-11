import { test, expect } from '@playwright/test'
import { interceptSearch, submitFromHome, trackAIRoutes, INFOBOX } from './fixtures'

test('intercepted query renders the thread in Sources-only mode', async ({ page }) => {
  await interceptSearch(page, { corrections: ['smoke correction'] })
  trackAIRoutes(page)

  await submitFromHome(page, 'smoke sources')
  await expect(page.getByRole('heading', { level: 1, name: /smoke sources/i })).toBeVisible()
  await expect(page.getByText('Alpha result')).toBeVisible()
  await expect(page.getByRole('group', { name: 'Did you mean' })).toBeVisible()
})

test('intercepted query renders the thread in Synthesized mode', async ({ page }) => {
  await interceptSearch(page, { infobox: INFOBOX })
  trackAIRoutes(page)

  await page.goto('/')
  await page.getByRole('button', { name: 'Synthesized answer' }).click()
  const input = page.locator('#folio-search')
  await input.fill('smoke synth')
  await input.press('Enter')

  await expect(page.getByRole('heading', { level: 1, name: /smoke synth/i })).toBeVisible()
  await expect(page.getByText('Alpha result')).toBeVisible()
  await expect(page.getByText('Stub synthesis.')).toBeVisible()
})
