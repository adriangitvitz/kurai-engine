export const SEARCH_INPUT_ID = 'folio-search'

export function asPageUrl(input: string): string | null {
  const text = input.trim()
  if (!text || /\s/.test(text)) return null
  const candidate = /^https?:\/\//i.test(text) ? text : /^www\./i.test(text) ? `https://${text}` : null
  if (!candidate) return null
  try {
    const url = new URL(candidate)
    return url.hostname.includes('.') ? url.href : null
  } catch {
    return null
  }
}

export function safeNext(next: string | null): string {
  return next && /^\/(?![/\\])/.test(next) ? next : '/'
}
