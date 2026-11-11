'use client'

import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore } from 'react'
import { flushSync } from 'react-dom'
import { search, synthesize, extractPage, runDeepResearch as runDeepResearchAPI, APIError } from './api'
import { SourceVM, Facet, facetModelFromCounts, facetKeyToContentType, toSource, cardSize, topUrls } from './folio'
import { Answers, SynthesizeResponse, DeepResearchStep, ExtractedArticle } from './types'
import { asPageUrl } from './nav'
import { SearchFilters, NO_FILTERS, parseSites, parseDate, splitSiteOperators, sameFilters } from './filters'
import { savedStore } from './savedStore'
import { answersStore } from './answersStore'

export type SearchMode = 'sources' | 'synth' | 'deep'
export type Language = 'en' | 'es'
type SynthState = 'none' | 'loading' | 'ready' | 'error'
type ResearchStatus = 'planning' | 'researching' | 'synthesizing' | 'ready' | 'error'
type LoadState = 'loading' | 'ready' | 'error'

export interface ReaderState {
  url: string
  status: LoadState
  article?: ExtractedArticle
  errorCode?: string
}

interface Entry {
  id: number
  query: string
  status: LoadState
  error?: string
  sources: SourceVM[]
  total: number
  page: number
  totalPages: number
  facet: string
  contentType: string
  facetCounts: Record<string, number>
  groupIndex: number
  loadingMore: boolean
  synth: SynthState
  answer?: SynthesizeResponse
  synthError?: string
  answers?: Answers
  reader?: ReaderState
  filters: SearchFilters
  timeRange: string
  phraseRelaxed: boolean
}

interface ResearchState {
  status: ResearchStatus
  query: string
  title: string
  overview: string
  metaTime: string
  metaSteps: number
  metaFacts: number
  steps: DeepResearchStep[]
  sources: SourceVM[]
  followups: string[]
  progress: { done: number; total: number } | null
  error: string
}

export interface CardVM {
  source: SourceVM
  size: ReturnType<typeof cardSize>
  hasImg: boolean
  saved: boolean
}

export interface EntryView {
  id: number
  query: string
  status: LoadState
  error?: string
  synth: SynthState
  answer?: SynthesizeResponse
  synthError?: string
  sourceCount: number
  total: number
  facet: string
  facets: Facet[]
  cards: CardVM[]
  groupIndex: number
  rangeStart: number
  groupCount: number
  hasMore: boolean
  canPrev: boolean
  canNext: boolean
  loadingMore: boolean
  showFollowups: boolean
  followupItems: string[]
  answers?: Answers
  reader?: ReaderState
  dateFiltered: boolean
  timeRange: string
  phraseRelaxed: boolean
  focusHeading: boolean
}

const GROUP_SIZE = 6
const SYNTH_URLS = 5

const MODES: SearchMode[] = ['sources', 'synth', 'deep']

interface UrlState {
  query: string
  mode: SearchMode
  pageUrl?: string
  filters: SearchFilters
}

function readUrlState(): UrlState {
  const params = new URLSearchParams(window.location.search)
  const mode = params.get('mode') as SearchMode
  return {
    query: (params.get('q') ?? '').trim(),
    mode: MODES.includes(mode) ? mode : 'sources',
    pageUrl: asPageUrl(params.get('url') ?? '') ?? undefined,
    filters: {
      sites: parseSites(params.get('site') ?? ''),
      from: parseDate(params.get('from')),
      to: parseDate(params.get('to')),
    },
  }
}

function urlFor({ query, mode, pageUrl, filters }: UrlState): string {
  if (pageUrl) return `/?${new URLSearchParams({ url: pageUrl })}`
  if (!query) return '/'
  const params = new URLSearchParams({ q: query })
  if (mode !== 'sources') params.set('mode', mode)
  if (filters.sites.length) params.set('site', filters.sites.join(','))
  if (filters.from) params.set('from', filters.from)
  if (filters.to) params.set('to', filters.to)
  return `/?${params}`
}

function groupCountOf(sources: SourceVM[]): number {
  return Math.max(1, Math.ceil(sources.length / GROUP_SIZE))
}

export function useFolioSearch() {
  const [view, setView] = useState<'home' | 'results' | 'research'>('home')
  const [searchMode, setSearchMode] = useState<SearchMode>('sources')
  const [language, setLanguageState] = useState<Language>('en')
  const [filters, setFilters] = useState<SearchFilters>(NO_FILTERS)
  const [thread, setThread] = useState<Entry[]>([])
  const [research, setResearch] = useState<ResearchState | null>(null)
  const [openSteps, setOpenSteps] = useState<Record<string, boolean>>({})
  const [boardOpen, setBoardOpen] = useState(false)
  const idc = useRef(1)

  const languageRef = useRef<Language>(language)

  const threadRef = useRef<Entry[]>(thread)
  useEffect(() => {
    threadRef.current = thread
  }, [thread])

  const saved = useSyncExternalStore(
    savedStore.subscribe,
    savedStore.getSnapshot,
    savedStore.getServerSnapshot
  )
  const toggleSave = useCallback(
    (source: SourceVM, query?: string) => void savedStore.toggle(source, query),
    []
  )

  useEffect(() => {
    void savedStore.hydrate()
  }, [])

  const answersOn = useSyncExternalStore(
    answersStore.subscribe,
    answersStore.getSnapshot,
    answersStore.getServerSnapshot
  )
  const [focusHeadingId, setFocusHeadingId] = useState<number | null>(null)

  const toggleAnswers = useCallback(() => {
    const headerAllowance = 80
    const els = Array.from(document.querySelectorAll<HTMLElement>('[data-entry-id]'))
    const anchor =
      els.find((el) => el.getBoundingClientRect().top >= headerAllowance) ??
      [...els].reverse().find((el) => el.getBoundingClientRect().bottom > headerAllowance)
    const before = anchor ? anchor.getBoundingClientRect().top : 0
    flushSync(() => answersStore.toggle())
    if (anchor) {
      const delta = anchor.getBoundingClientRect().top - before
      if (delta !== 0) window.scrollBy(0, delta)
    }
  }, [])

  const patchEntry = useCallback((id: number, patch: Partial<Entry> | ((e: Entry) => Partial<Entry>)) => {
    setThread((t) =>
      t.map((e) => (e.id === id ? { ...e, ...(typeof patch === 'function' ? patch(e) : patch) } : e))
    )
  }, [])

  const runSynth = useCallback(
    async (id: number, sources: SourceVM[], query: string) => {
      const urls = topUrls(sources, SYNTH_URLS)
      if (urls.length === 0) {
        patchEntry(id, { synth: 'error', synthError: 'No sources to read' })
        return
      }
      patchEntry(id, { synth: 'loading', synthError: undefined })
      try {
        const answer = await synthesize(query, urls, languageRef.current)
        patchEntry(id, { synth: 'ready', answer })
      } catch (err) {
        const msg =
          err instanceof APIError && err.status === 503
            ? 'Synthesis isn’t configured on this server.'
            : 'Couldn’t synthesize an answer. The sources are still below.'
        patchEntry(id, { synth: 'error', synthError: msg })
      }
    },
    [patchEntry]
  )

  const startResearch = useCallback(async (query: string) => {
    setView('research')
    setResearch({
      status: 'planning',
      query,
      title: '',
      overview: '',
      metaTime: '',
      metaSteps: 0,
      metaFacts: 0,
      steps: [],
      sources: [],
      followups: [],
      progress: null,
      error: '',
    })
    setOpenSteps({})
    try {
      await runDeepResearchAPI(query, languageRef.current, (event) => {
        switch (event.type) {
          case 'plan': {
            const steps: DeepResearchStep[] = event.steps.map((s, i) => ({
              id: `step-${i}`,
              n: String(i + 1).padStart(2, '0'),
              title: s.title,
              summary: s.summary,
              time: '',
              isStart: i === 0,
              desc: '',
              facts: [],
            }))
            setResearch((r) =>
              r && {
                ...r,
                status: 'researching',
                title: event.title,
                overview: event.overview,
                metaSteps: steps.length,
                steps,
              }
            )
            if (steps[0]) setOpenSteps({ [steps[0].id]: true })
            break
          }
          case 'reading':
            setResearch((r) => r && { ...r, progress: { done: event.done, total: event.total } })
            break
          case 'synthesizing':
            setResearch((r) => r && { ...r, status: 'synthesizing' })
            break
          case 'done': {
            const sources = event.result.sources.map((s, i) => toSource(s, i))
            setResearch(
              (r) =>
                r && {
                  ...r,
                  status: 'ready',
                  title: event.result.title,
                  overview: event.result.overview,
                  metaTime: event.result.metaTime,
                  metaSteps: event.result.metaSteps,
                  metaFacts: event.result.metaFacts,
                  steps: event.result.steps,
                  sources,
                  followups: event.result.followups,
                }
            )
            break
          }
          case 'error':
            setResearch((r) => r && { ...r, status: 'error', error: event.message })
            break
        }
      })
    } catch (err) {
      const msg =
        err instanceof APIError && err.status === 503
          ? 'Deep research isn’t configured on this server.'
          : 'Deep research failed. Try again in a moment.'
      setResearch((r) => r && { ...r, status: 'error', error: msg })
    }
  }, [])

  const toggleStep = useCallback((id: string) => {
    setOpenSteps((s) => ({ ...s, [id]: !s[id] }))
  }, [])

  const readPage = useCallback(
    async (url: string, append = false) => {
      const id = idc.current++
      const entry: Entry = {
        id,
        query: url,
        status: 'ready',
        sources: [],
        total: 0,
        page: 1,
        totalPages: 1,
        facet: 'All',
        contentType: '',
        facetCounts: {},
        groupIndex: 0,
        loadingMore: false,
        synth: 'none',
        reader: { url, status: 'loading' },
        filters: NO_FILTERS,
        timeRange: '',
        phraseRelaxed: false,
      }
      setView('results')
      setThread((t) => (append ? [...t, entry] : [entry]))
      if (append) setFocusHeadingId(id)
      try {
        const article = await extractPage(url)
        patchEntry(id, { reader: { url, status: 'ready', article } })
      } catch (err) {
        patchEntry(id, {
          reader: { url, status: 'error', errorCode: err instanceof APIError ? err.code : undefined },
        })
      }
    },
    [patchEntry]
  )

  const submitQuery = useCallback(
    async (raw: string, append = false, forceMode?: SearchMode, forceFilters?: SearchFilters) => {
      const typed = (raw || '').trim()
      if (!typed) return
      const pageUrl = asPageUrl(typed)
      if (pageUrl) {
        void readPage(pageUrl, append)
        return
      }
      const mode = forceMode ?? searchMode

      if (mode === 'deep') {
        startResearch(typed)
        return
      }

      const operators = splitSiteOperators(typed)
      const query = operators.query
      if (!query) return
      const base = forceFilters ?? filters
      const active = operators.sites.length ? { ...base, sites: operators.sites } : base
      if (!sameFilters(active, filters)) setFilters(active)

      const id = idc.current++
      const wantSynth = mode === 'synth'
      const entry: Entry = {
        id,
        query,
        status: 'loading',
        sources: [],
        total: 0,
        page: 1,
        totalPages: 1,
        facet: 'All',
        contentType: '',
        facetCounts: {},
        groupIndex: 0,
        loadingMore: false,
        synth: wantSynth ? 'loading' : 'none',
        filters: active,
        timeRange: '',
        phraseRelaxed: false,
      }

      setView('results')
      setThread((t) => (append ? [...t, entry] : [entry]))

      try {
        const resp = await search(query, { filters: active })
        const sources = (resp.results ?? []).map(toSource)
        patchEntry(id, {
          status: 'ready',
          sources,
          total: resp.meta?.total ?? sources.length,
          page: resp.meta?.page ?? 1,
          totalPages: resp.meta?.total_pages ?? 1,
          facetCounts: resp.meta?.facets ?? {},
          answers: resp.answers,
          timeRange: resp.meta?.time_range ?? '',
          phraseRelaxed: !!resp.meta?.phrase_relaxed,
        })
        if (wantSynth) runSynth(id, sources, query)
      } catch (err) {
        const msg = err instanceof APIError ? err.message : 'Search failed'
        patchEntry(id, { status: 'error', error: msg, synth: 'none' })
      }
    },
    [searchMode, filters, patchEntry, runSynth, startResearch, readPage]
  )

  const submitCorrection = useCallback(
    (corrected: string) => {
      const newId = idc.current
      setFocusHeadingId(newId)
      void submitQuery(corrected, true)
    },
    [submitQuery]
  )

  const setMode = useCallback(
    (m: SearchMode) => {
      setSearchMode(m)
      if (view === 'home') return

      const last = thread[thread.length - 1]
      if (view === 'results' && last?.reader) return

      if (m === 'deep') {
        const q = view === 'research' ? research?.query : last?.query
        if (q) startResearch(q)
        return
      }

      if (view === 'research') {
        if (research?.query) submitQuery(research.query, false, m)
        return
      }

      if (last && last.status === 'ready' && m === 'synth' && last.synth === 'none') {
        runSynth(last.id, last.sources, last.query)
      }
    },
    [view, research, thread, runSynth, startResearch, submitQuery]
  )

  const setLanguage = useCallback(
    (lang: Language) => {
      languageRef.current = lang
      setLanguageState(lang)
      if (research && (research.status === 'ready' || research.status === 'error')) {
        startResearch(research.query)
        return
      }
      const last = threadRef.current[threadRef.current.length - 1]
      if (last && last.status === 'ready' && (last.synth === 'ready' || last.synth === 'error')) {
        runSynth(last.id, last.sources, last.query)
      }
    },
    [research, runSynth, startResearch]
  )

  const selectFacet = useCallback(
    async (id: number, facetKey: string) => {
      const e = threadRef.current.find((x) => x.id === id)
      if (!e || facetKey === e.facet) return
      const contentType = facetKeyToContentType(facetKey)
      patchEntry(id, { facet: facetKey, contentType, groupIndex: 0, loadingMore: true })
      try {
        const resp = await search(e.query, { contentType, filters: e.filters })
        const sources = (resp.results ?? []).map(toSource)
        patchEntry(id, {
          sources,
          total: resp.meta?.total ?? sources.length,
          page: resp.meta?.page ?? 1,
          totalPages: resp.meta?.total_pages ?? 1,
          facetCounts: resp.meta?.facets ?? e.facetCounts,
          loadingMore: false,
        })
      } catch {
        patchEntry(id, { loadingMore: false })
      }
    },
    [patchEntry]
  )

  const nextGroup = useCallback(
    async (id: number) => {
      const e = threadRef.current.find((x) => x.id === id)
      if (!e || e.loadingMore) return

      const atEnd = e.groupIndex >= groupCountOf(e.sources) - 1
      const hasMore = e.page < e.totalPages

      if (atEnd && hasMore) {
        patchEntry(id, { loadingMore: true })
        try {
          const resp = await search(e.query, { page: e.page + 1, contentType: e.contentType, filters: e.filters })
          const incoming = (resp.results ?? []).map(toSource)
          const seen = new Set(e.sources.map((s) => s.id))
          const merged = [...e.sources, ...incoming.filter((s) => !seen.has(s.id))]
          patchEntry(id, {
            sources: merged,
            page: resp.meta?.page ?? e.page + 1,
            totalPages: resp.meta?.total_pages ?? e.totalPages,
            total: resp.meta?.total ?? e.total,
            facetCounts: resp.meta?.facets ?? e.facetCounts,
            loadingMore: false,
            groupIndex: Math.min(e.groupIndex + 1, groupCountOf(merged) - 1),
          })
        } catch {
          patchEntry(id, { loadingMore: false })
        }
        return
      }

      if (!atEnd) {
        patchEntry(id, (cur) => ({
          groupIndex: Math.min(cur.groupIndex + 1, groupCountOf(cur.sources) - 1),
        }))
      }
    },
    [patchEntry]
  )

  const prevGroup = useCallback(
    (id: number) => patchEntry(id, (e) => ({ groupIndex: Math.max(0, e.groupIndex - 1) })),
    [patchEntry]
  )

  const goHome = useCallback(() => setView('home'), [])

  const lastEntry = view === 'results' ? thread[thread.length - 1] : undefined
  const stateUrl = urlFor({
    query: view === 'research' ? (research?.query ?? '') : (lastEntry?.query ?? ''),
    mode: searchMode,
    pageUrl: lastEntry?.reader?.url,
    filters: view === 'results' ? filters : NO_FILTERS,
  })

  const applyFilters = useCallback(
    (next: SearchFilters) => {
      setFilters(next)
      const last = thread[thread.length - 1]
      if (view === 'results' && last && !last.reader) void submitQuery(last.query, false, undefined, next)
    },
    [view, thread, submitQuery]
  )
  const appliedUrl = useRef('/')
  const pushedStateUrl = useRef('/')

  const applyUrl = useCallback(() => {
    const next = readUrlState()
    if (urlFor(next) === appliedUrl.current) return
    appliedUrl.current = urlFor(next)
    if (next.pageUrl) {
      void readPage(next.pageUrl)
      return
    }
    if (!next.query) {
      setView('home')
      return
    }
    setSearchMode(next.mode)
    setFilters(next.filters)
    void submitQuery(next.query, false, next.mode, next.filters)
  }, [submitQuery, readPage])

  const applyUrlRef = useRef(applyUrl)
  useEffect(() => {
    applyUrlRef.current = applyUrl
  }, [applyUrl])

  useEffect(() => {
    const onPopState = () => applyUrlRef.current()
    onPopState()
    window.addEventListener('popstate', onPopState)
    return () => window.removeEventListener('popstate', onPopState)
  }, [])

  useEffect(() => {
    if (stateUrl === pushedStateUrl.current) return
    pushedStateUrl.current = stateUrl
    appliedUrl.current = stateUrl
    if (stateUrl !== urlFor(readUrlState())) window.history.pushState(null, '', stateUrl)
  }, [stateUrl])

  const entries: EntryView[] = useMemo(
    () =>
      thread.map((e) => {
        const facets = facetModelFromCounts(e.facetCounts)
        const groupCount = groupCountOf(e.sources)
        const gi = Math.min(e.groupIndex, groupCount - 1)
        const start = gi * GROUP_SIZE
        const cards: CardVM[] = e.sources.slice(start, start + GROUP_SIZE).map((source, i) => {
          const size = cardSize(i)
          return { source, size, hasImg: size !== 'compact', saved: !!saved[source.id] }
        })
        const hasMore = e.page < e.totalPages
        return {
          id: e.id,
          query: e.query,
          status: e.status,
          error: e.error,
          synth: e.synth,
          answer: e.answer,
          synthError: e.synthError,
          sourceCount: e.sources.length,
          total: e.total || e.sources.length,
          facet: e.facet,
          facets,
          cards,
          groupIndex: gi,
          rangeStart: start + 1,
          groupCount,
          hasMore,
          canPrev: gi > 0,
          canNext: gi < groupCount - 1 || hasMore,
          loadingMore: e.loadingMore,
          showFollowups: searchMode === 'synth' && e.synth === 'ready',
          followupItems: e.answer?.followups ?? [],
          answers: e.answers,
          reader: e.reader,
          dateFiltered: !!(e.filters.from || e.filters.to),
          timeRange: e.timeRange,
          phraseRelaxed: e.phraseRelaxed,
          focusHeading: e.id === focusHeadingId,
        }
      }),
    [thread, saved, searchMode, focusHeadingId]
  )

  const savedList = useMemo(() => Object.values(saved), [saved])

  const researchSources = useMemo(
    () => (research ? research.sources.map((s) => ({ ...s, saved: !!saved[s.id] })) : []),
    [research, saved]
  )

  const modeHint =
    searchMode === 'synth'
      ? 'Reads sources & writes an answer · uses AI'
      : searchMode === 'deep'
        ? 'Builds a focus-friendly path to learn a topic · uses AI'
        : 'Fast · no AI · results only'

  return {
    view,
    searchMode,
    language,
    setLanguage,
    entries,
    savedList,
    boardOpen,
    modeHint,
    research,
    researchSources,
    openSteps,
    toggleStep,
    startResearch,
    submitQuery,
    readPage,
    filters,
    applyFilters,
    submitCorrection,
    answersOn,
    toggleAnswers,
    setMode,
    goHome,
    toggleSave,
    selectFacet,
    nextGroup,
    prevGroup,
    runSynth: (id: number) => {
      const e = thread.find((x) => x.id === id)
      if (e) runSynth(id, e.sources, e.query)
    },
    openBoard: () => setBoardOpen(true),
    closeBoard: () => setBoardOpen(false),
    toggleBoard: () => setBoardOpen((o) => !o),
  }
}
