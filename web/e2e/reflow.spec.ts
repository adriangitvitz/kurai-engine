import { test, expect } from '@playwright/test'
import { interceptSearch, trackAIRoutes, CORRECTIONS } from './fixtures'

test.use({ viewport: { width: 320, height: 800 } })

test('corrections line and header reflow at 320px without horizontal scroll', async ({ page }) => {
  await interceptSearch(page, { corrections: CORRECTIONS })
  trackAIRoutes(page)

  await page.goto('/')
  const input = page.locator('#folio-search')
  await input.fill('mechanicl keyboards')
  await input.press('Enter')

  await expect(page.getByRole('group', { name: 'Did you mean' })).toBeVisible()

  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth
  )
  expect(overflow, 'horizontal overflow in CSS px').toBeLessThanOrEqual(0)

  await expect(page.getByRole('switch', { name: 'Answer cards & panels' })).toBeVisible()
  for (const c of CORRECTIONS) {
    await expect(page.getByRole('button', { name: c })).toBeVisible()
  }
})
