export interface SearchFilters {
  sites: string[]
  from: string
  to: string
}

export const NO_FILTERS: SearchFilters = { sites: [], from: '', to: '' }

const DATE = /^\d{4}-\d{2}-\d{2}$/
const SITE_OPERATOR = /(^|\s)site:(\S+)/gi

export function parseSites(text: string): string[] {
  const sites = text
    .split(/[\s,]+/)
    .map((s) =>
      s
        .toLowerCase()
        .replace(/^https?:\/\//, '')
        .replace(/^www\./, '')
        .replace(/\/.*$/, '')
    )
    .filter((s) => /^[a-z0-9]([a-z0-9.-]*[a-z0-9])?$/.test(s))
  return [...new Set(sites)]
}

export function parseDate(text: string | null): string {
  return text && DATE.test(text) ? text : ''
}

export function splitSiteOperators(query: string): { query: string; sites: string[] } {
  const found: string[] = []
  const rest = query.replace(SITE_OPERATOR, (_match, lead: string, site: string) => {
    found.push(site)
    return lead
  })
  return { query: rest.replace(/\s+/g, ' ').trim(), sites: parseSites(found.join(' ')) }
}

export function hasFilters(f: SearchFilters): boolean {
  return f.sites.length > 0 || !!f.from || !!f.to
}

export function sameFilters(a: SearchFilters, b: SearchFilters): boolean {
  return a.from === b.from && a.to === b.to && a.sites.join(',') === b.sites.join(',')
}
