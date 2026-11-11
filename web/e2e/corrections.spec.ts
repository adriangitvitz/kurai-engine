import { test, expect } from '@playwright/test'
import { interceptSearch, submitFromHome, trackAIRoutes, CORRECTIONS } from './fixtures'

for (const mode of ['sources', 'synth'] as const) {
  test(`correction line renders and re-queries (${mode} mode)`, async ({ page }) => {
    await interceptSearch(page, { corrections: CORRECTIONS })
    const ai = trackAIRoutes(page)

    await page.goto('/')
    if (mode === 'synth') await page.getByRole('button', { name: 'Synthesized answer' }).click()
    const input = page.locator('#folio-search')
    await input.fill('mechanicl keyboards')
    await input.press('Enter')

    const line = page.getByRole('group', { name: 'Did you mean' })
    await expect(line).toBeVisible()
    await expect(line).toHaveCount(1)

    await line.getByRole('button', { name: CORRECTIONS[0] }).click()
    await expect(page.getByRole('heading', { level: 1, name: /mechanical keyboards/i })).toBeVisible()

    if (mode === 'sources') {
      expect(ai.synthesize, 'sources-only stays no-AI').toBe(0)
      expect(ai.deepResearch).toBe(0)
    } else {
      await expect(page.getByText('Stub synthesis.').last()).toBeVisible()
    }
  })
}

test('focus moves to the new entry heading after correction', async ({ page }) => {
  await interceptSearch(page, { corrections: CORRECTIONS })
  trackAIRoutes(page)

  await submitFromHome(page, 'mechanicl keyboards')
  const line = page.getByRole('group', { name: 'Did you mean' })
  await line.getByRole('button', { name: CORRECTIONS[0] }).click()

  const newHeading = page.getByRole('heading', { level: 1, name: /mechanical keyboards/i })
  await expect(newHeading).toBeVisible()
  await expect(newHeading).toBeFocused()
})

test('corrections line is exempt from the display preference', async ({ page }) => {
  await interceptSearch(page, { corrections: CORRECTIONS })
  trackAIRoutes(page)

  await submitFromHome(page, 'mechanicl keyboards')
  await expect(page.getByRole('group', { name: 'Did you mean' })).toBeVisible()

  await page.getByRole('switch', { name: 'Answer cards & panels' }).click()
  await expect(page.getByRole('group', { name: 'Did you mean' })).toBeVisible()
})
