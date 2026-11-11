import type { SourceVM } from './folio'
import { listBookmarks, saveBookmark, deleteBookmark, APIError } from './api'
import type { BookmarkVM } from './types'

const LEGACY_KEY = 'folio.saved.v1'
const IMPORTED_FLAG = 'folio.saved.imported.v1'
const EMPTY: Record<string, SourceVM> = {}

let current: Record<string, SourceVM> = EMPTY
let hydrated = false
let hydrating: Promise<void> | null = null
const listeners = new Set<() => void>()

function emit() {
  for (const l of listeners) l()
}

function toSourceVM(b: BookmarkVM): SourceVM {
  return {
    id: b.url,
    url: b.url,
    title: b.title,
    snippet: b.snippet ?? '',
    type: b.content_type ?? '',
    rawType: (b.content_type ?? '') as SourceVM['rawType'],
    domain: b.domain ?? '',
    meta: b.meta ?? '',
    score: b.score ?? 0,
    size: 'compact',
    hasImg: false,
    imgLabel: b.content_type ?? '',
    query: b.query,
    savedAt: b.saved_at,
  }
}

function toInput(s: SourceVM, query?: string) {
  return {
    url: s.url,
    title: s.title,
    snippet: s.snippet,
    domain: s.domain,
    content_type: s.rawType,
    meta: s.meta,
    score: s.score,
    query,
  }
}

export const savedStore = {
  subscribe(cb: () => void) {
    listeners.add(cb)
    return () => {
      listeners.delete(cb)
    }
  },
  getSnapshot(): Record<string, SourceVM> {
    return current
  },
  getServerSnapshot(): Record<string, SourceVM> {
    return EMPTY
  },

  hydrate(): Promise<void> {
    if (hydrated) return Promise.resolve()
    if (hydrating) return hydrating

    hydrating = (async () => {
      try {
        await importLegacy()
        const list = await listBookmarks()
        const next: Record<string, SourceVM> = {}
        for (const b of list) next[b.url] = toSourceVM(b)
        current = next
        hydrated = true
        emit()
      } catch (err) {
        if (!(err instanceof APIError) || err.status !== 401) {
          console.warn('saved board: failed to load', err)
        }
      } finally {
        hydrating = null
      }
    })()
    return hydrating
  },

  async toggle(source: SourceVM, query?: string) {
    const previous = current
    const saving = !current[source.id]

    if (saving) {
      current = { [source.id]: { ...source, query, savedAt: new Date().toISOString() }, ...previous }
    } else {
      const next = { ...previous }
      delete next[source.id]
      current = next
    }
    emit()

    try {
      if (saving) {
        const stored = await saveBookmark(toInput(source, query))
        current = { ...current, [source.id]: toSourceVM(stored) }
        emit()
      } else {
        await deleteBookmark(source.url)
      }
    } catch (err) {
      current = previous
      emit()
      console.warn('saved board: write failed, rolled back', err)
    }
  },
}

async function importLegacy(): Promise<void> {
  if (typeof localStorage === 'undefined') return
  if (localStorage.getItem(IMPORTED_FLAG)) return

  let legacy: Record<string, SourceVM>
  try {
    const raw = localStorage.getItem(LEGACY_KEY)
    if (!raw) {
      localStorage.setItem(IMPORTED_FLAG, '1')
      return
    }
    legacy = JSON.parse(raw) as Record<string, SourceVM>
  } catch {
    localStorage.setItem(IMPORTED_FLAG, '1')
    return
  }

  const entries = Object.values(legacy)
  if (entries.length === 0) {
    localStorage.setItem(IMPORTED_FLAG, '1')
    return
  }

  for (const s of entries) {
    await saveBookmark(toInput(s))
  }
  localStorage.setItem(IMPORTED_FLAG, '1')
}
