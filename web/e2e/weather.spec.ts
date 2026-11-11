import { test, expect } from '@playwright/test'
import { interceptSearch, submitFromHome, trackAIRoutes, WEATHER } from './fixtures'

const OAXACA_CARD = { name: 'Weather: Oaxaca City, Mexico' }

const SPRINGFIELD = {
  ...WEATHER,
  place: 'Springfield, Missouri, United States',
  country_code: 'US',
  timezone: 'America/Chicago',
  temperature_c: 20,
  as_of: '2026-07-17T14:30:00-05:00',
}

const REYKJAVIK = {
  ...WEATHER,
  place: 'Reykjavik, Iceland',
  country_code: 'IS',
  timezone: 'Atlantic/Reykjavik',
  temperature_c: 11.4,
  as_of: '2026-07-17T19:30:00+00:00',
}

test('card shows all required content as text', async ({ page }) => {
  await interceptSearch(page, { weather: WEATHER })
  trackAIRoutes(page)
  await submitFromHome(page, 'weather in oaxaca')

  const card = page.getByRole('region', OAXACA_CARD)
  await expect(card).toBeVisible()
  await expect(card).toContainText('Oaxaca City, Mexico')
  await expect(card).toContainText('24°C')
  await expect(card).toContainText('Partly cloudy')
  await expect(card).toContainText(/open-meteo/i)
  await expect(card).toContainText(/as of/i)
  await expect(card).toContainText('Fri')
})

test('non-US place shows metric only', async ({ page }) => {
  await interceptSearch(page, { weather: REYKJAVIK })
  trackAIRoutes(page)
  await submitFromHome(page, 'weather in reykjavik')

  const card = page.getByRole('region', { name: 'Weather: Reykjavik, Iceland' })
  await expect(card).toBeVisible()
  await expect(card).toContainText('11°C')
  const text = await card.textContent()
  expect(text).not.toContain('°F')
})

test('US place shows Fahrenheit alongside Celsius, binding place exemplar verbatim', async ({ page }) => {
  await interceptSearch(page, { weather: SPRINGFIELD })
  trackAIRoutes(page)
  await submitFromHome(page, 'weather in springfield')

  const card = page.getByRole('region', { name: 'Weather: Springfield, Missouri, United States' })
  await expect(card).toBeVisible()
  await expect(card).toContainText('Springfield, Missouri, United States')
  await expect(card).toContainText('20°C / 68°F')
})

test('as-of time renders place-local with timezone label and data-asof', async ({ page }) => {
  await interceptSearch(page, { weather: WEATHER })
  trackAIRoutes(page)
  await submitFromHome(page, 'weather in oaxaca')

  const card = page.getByRole('region', OAXACA_CARD)
  await expect(card).toContainText('As of 14:00 (America/Mexico_City)')
  await expect(card.locator('[data-asof]')).toHaveAttribute('data-asof', '2026-07-17T14:00:00-06:00')
})

test('the Open-Meteo credit link is keyboard-reachable in reading order', async ({ page }) => {
  await interceptSearch(page, { weather: WEATHER })
  trackAIRoutes(page)
  await submitFromHome(page, 'weather in oaxaca')

  const card = page.getByRole('region', OAXACA_CARD)
  const credit = card.getByRole('link', { name: /open-meteo/i })
  await expect(credit).toBeVisible()

  await credit.focus()
  await expect(credit).toBeFocused()
  await expect(credit).toHaveAttribute('href', 'https://open-meteo.com/')
})

test('weather card reflows at 320px — forecast wraps, values keep units', async ({ browser }) => {
  const ctx = await browser.newContext({ viewport: { width: 320, height: 800 } })
  const p = await ctx.newPage()
  await interceptSearch(p, { weather: WEATHER })
  trackAIRoutes(p)
  await p.goto('/')
  const input = p.locator('#folio-search')
  await input.fill('weather in oaxaca')
  await input.press('Enter')

  const card = p.getByRole('region', OAXACA_CARD)
  await expect(card).toBeVisible()
  await expect(card).toContainText('°C')
  await expect(card).toContainText('Oaxaca City, Mexico')

  const overflow = await p.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth
  )
  expect(overflow).toBeLessThanOrEqual(0)
  await ctx.close()
})
