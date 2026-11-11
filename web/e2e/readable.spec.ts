import { test, expect, type Page } from '@playwright/test'
import { interceptSearch, submitFromHome, trackAIRoutes, INFOBOX, WEATHER } from './fixtures'

const CONTRAST_FN = `
  (el) => {
    const parse = (c) => {
      const m = c.match(/rgba?\\(([^)]+)\\)/);
      if (!m) return null;
      const p = m[1].split(',').map((x) => parseFloat(x));
      return { r: p[0], g: p[1], b: p[2], a: p.length > 3 ? p[3] : 1 };
    };
    const lin = (v) => { v /= 255; return v <= 0.04045 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); };
    const lum = (c) => 0.2126 * lin(c.r) + 0.7152 * lin(c.g) + 0.0722 * lin(c.b);
    const bgOf = (node) => {
      let n = node;
      while (n) {
        const c = parse(getComputedStyle(n).backgroundColor);
        if (c && c.a > 0.95) return c;
        n = n.parentElement;
      }
      return { r: 23, g: 23, b: 22, a: 1 };
    };
    const cs = getComputedStyle(el);
    const fg = parse(cs.color);
    const bg = bgOf(el);
    if (!fg) return null;
    let opacity = 1, n = el;
    while (n) { opacity *= parseFloat(getComputedStyle(n).opacity || '1'); n = n.parentElement; }
    if (opacity < 0.05) return null;
    const eff = {
      r: fg.r * opacity + bg.r * (1 - opacity),
      g: fg.g * opacity + bg.g * (1 - opacity),
      b: fg.b * opacity + bg.b * (1 - opacity),
    };
    const [hi, lo] = [lum(eff), lum(bg)].sort((a, b) => b - a);
    return {
      ratio: (hi + 0.05) / (lo + 0.05),
      size: parseFloat(cs.fontSize),
      weight: cs.fontWeight,
      text: (el.textContent || '').trim().slice(0, 40),
    };
  }
`

async function textMetrics(page: Page) {
  return page.$$eval(
    'body *',
    (els, fn) => {
      const measure = eval(fn) as (el: Element) => unknown
      const NON_CONTENT = ['STYLE', 'SCRIPT', 'NOSCRIPT', 'TITLE', 'NEXTJS-PORTAL']
      return els
        .filter((el) => {
          if (NON_CONTENT.includes(el.tagName)) return false
          if (el.closest('nextjs-portal, [data-nextjs-toast], [data-nextjs-dialog]')) return false
          const own = Array.from(el.childNodes).some(
            (n) => n.nodeType === 3 && (n.textContent || '').trim().length > 1
          )
          if (!own) return false
          const r = el.getBoundingClientRect()
          const cs = getComputedStyle(el)
          return r.width > 0 && r.height > 0 && cs.visibility !== 'hidden' && cs.display !== 'none'
        })
        .map(measure)
        .filter(Boolean)
    },
    CONTRAST_FN
  ) as Promise<{ ratio: number; size: number; weight: string; text: string }[]>
}

async function resultsWithEverything(page: Page) {
  await interceptSearch(page, { infobox: INFOBOX, weather: WEATHER, corrections: ['did you mean this'] })
  trackAIRoutes(page)
  await submitFromHome(page, 'clima en guadalajara')
  await expect(page.getByRole('region', { name: /Weather:/ })).toBeVisible()
  await page.evaluate(() =>
    Promise.all(document.getAnimations().map((a) => a.finished.catch(() => {})))
  )
}

test('no rendered text falls below the AA contrast floor', async ({ page }) => {
  await resultsWithEverything(page)
  const failures = (await textMetrics(page)).filter((m) => m.ratio < 4.5)
  expect(
    failures,
    `text below 4.5:1:\n${failures.map((f) => `  ${f.ratio.toFixed(2)}:1 "${f.text}"`).join('\n')}`
  ).toEqual([])
})

test('secondary text renders at AAA contrast in a warm non-gray', async ({ page }) => {
  await resultsWithEverything(page)

  const domain = page.locator('text=/alpha\\.example/').first()
  const m = (await domain.evaluate(eval(CONTRAST_FN) as never)) as { ratio: number }
  expect(m.ratio).toBeGreaterThanOrEqual(7)

  const color = await domain.evaluate((el) => getComputedStyle(el).color)
  const [r, g, b] = color.match(/\d+/g)!.map(Number)
  expect(r - b, `expected a warm colour, got ${color}`).toBeGreaterThan(12)
})

test('no text renders below 13px at the default font size', async ({ page }) => {
  await resultsWithEverything(page)
  const tooSmall = (await textMetrics(page)).filter((m) => m.size < 13)
  expect(
    tooSmall,
    `text below the 13px floor:\n${tooSmall.map((f) => `  ${f.size}px "${f.text}"`).join('\n')}`
  ).toEqual([])
})

test('text scales with the browser font-size preference', async ({ page }) => {
  await resultsWithEverything(page)
  const before = (await textMetrics(page)).map((m) => m.size)

  await page.evaluate(() => {
    document.documentElement.style.fontSize = '24px'
  })
  await page.evaluate(() =>
    Promise.all(document.getAnimations().map((a) => a.finished.catch(() => {})))
  )
  await page.waitForTimeout(300)
  const after = (await textMetrics(page)).map((m) => m.size)

  const grew = after.filter((s, i) => before[i] && s > before[i] + 0.5).length
  expect(grew, 'no text responded to the root font size — sizes are still px').toBeGreaterThan(5)

  const metaBefore = Math.min(...before)
  const metaAfter = Math.min(...after)
  expect(metaAfter / metaBefore).toBeGreaterThan(1.3)
})

test('the mono 600 weight is a real font face, not synthesized', async ({ page }) => {
  await resultsWithEverything(page)
  const loaded = await page.evaluate(async () => {
    await document.fonts.ready
    return {
      has600: document.fonts.check("600 13px 'IBM Plex Mono'"),
      faces: [...document.fonts]
        .filter((f) => f.family.includes('Plex'))
        .map((f) => `${f.weight}`),
    }
  })
  expect(loaded.has600, 'weight 600 would be synthesized — smeared strokes').toBe(true)
  expect(loaded.faces, 'a real 600 face must be registered').toContain('600')
})

test('the content column uses the available width and matches the header', async ({ browser }) => {
  const ctx = await browser.newContext({ viewport: { width: 1600, height: 900 } })
  const page = await ctx.newPage()
  await resultsWithEverything(page)

  const widths = await page.evaluate(() => {
    const h1 = document.querySelector('h1')
    const column = h1?.closest('div[style*="max-width"]') as HTMLElement | null
    const header = document.querySelector('div[style*="sticky"] > div') as HTMLElement | null
    return { column: column?.getBoundingClientRect().width ?? 0, header: header?.getBoundingClientRect().width ?? 0 }
  })
  expect(widths.column).toBeGreaterThanOrEqual(1400)
  expect(widths.header).toBeGreaterThanOrEqual(1400)
  expect(widths.column).toBeLessThan(1600)
  expect(Math.abs(widths.column - widths.header)).toBeLessThan(2)
  await ctx.close()
})

test('320px reflows with the larger type', async ({ browser }) => {
  const ctx = await browser.newContext({ viewport: { width: 320, height: 800 } })
  const page = await ctx.newPage()
  await resultsWithEverything(page)

  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth
  )
  expect(overflow, 'horizontal overflow at 320px').toBeLessThanOrEqual(0)
  await ctx.close()
})

test('320px still reflows when the user has enlarged their font size', async ({ browser }) => {
  const ctx = await browser.newContext({ viewport: { width: 320, height: 800 } })
  const page = await ctx.newPage()
  await resultsWithEverything(page)
  await page.evaluate(() => {
    document.documentElement.style.fontSize = '24px'
  })
  await page.waitForTimeout(400)

  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth
  )
  expect(overflow, 'enlarged text must still reflow, not scroll sideways').toBeLessThanOrEqual(0)
  await ctx.close()
})
