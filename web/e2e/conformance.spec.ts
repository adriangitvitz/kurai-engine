import { test, expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { RESULTS, META, CURRENCY, WEATHER, INFOBOX } from './fixtures'

function collectKeys(value: unknown, out: Set<string>) {
  if (Array.isArray(value)) {
    for (const item of value) collectKeys(item, out)
  } else if (value && typeof value === 'object') {
    for (const [k, v] of Object.entries(value)) {
      out.add(k)
      collectKeys(v, out)
    }
  }
}

test('every fixture field name appears in the OpenAPI contract', async () => {
  const doc = readFileSync(join(__dirname, '..', '..', 'internal', 'server', 'openapi.yaml'), 'utf8')
  const keys = new Set<string>()
  collectKeys({ results: RESULTS, meta: META, answers: { currency: CURRENCY, weather: WEATHER, infobox: INFOBOX } }, keys)

  const missing = [...keys].filter((k) => !doc.includes(k))
  expect(missing, `fixture fields absent from the OpenAPI document: ${missing.join(', ')}`).toEqual([])
})
