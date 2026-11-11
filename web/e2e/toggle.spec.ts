import { test, expect } from '@playwright/test'
import { interceptSearch, submitFromHome, trackAIRoutes } from './fixtures'

test('switch exposes name, role, state and operates by keyboard', async ({ page }) => {
  await interceptSearch(page)
  trackAIRoutes(page)
  await submitFromHome(page, 'any query')

  const sw = page.getByRole('switch', { name: 'Answer cards & panels' })
  await expect(sw).toBeVisible()
  await expect(sw).toHaveAttribute('aria-checked', 'true')

  await sw.focus()
  await page.keyboard.press('Space')
  await expect(sw).toHaveAttribute('aria-checked', 'false')
  await expect(sw).toBeFocused()

  await page.keyboard.press('Enter')
  await expect(sw).toHaveAttribute('aria-checked', 'true')
  await expect(sw).toBeFocused()

  await sw.focus()
  const outline = await sw.evaluate((el) => getComputedStyle(el).outlineStyle)
  expect(outline).not.toBe('none')
})
