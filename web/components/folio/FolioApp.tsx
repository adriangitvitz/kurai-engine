'use client'

import { useEffect } from 'react'
import { useFolioSearch } from '@/lib/useFolioSearch'
import { CONTENT_WIDTH } from '@/lib/layout'
import { SEARCH_INPUT_ID } from '@/lib/nav'
import { HomeHero } from './HomeHero'
import { ResultsHeader } from './ResultsHeader'
import { QueryEntry } from './QueryEntry'
import { PageEntry } from './PageEntry'
import { FilterRow } from './FilterRow'
import { ResearchView } from './ResearchView'
import { BoardDrawer } from './BoardDrawer'

export function FolioApp() {
  const f = useFolioSearch()

  const activeQuery = f.entries[f.entries.length - 1]?.query ?? ''

  useEffect(() => {
    const focusSearch = (e: KeyboardEvent) => {
      if (e.key !== '/' || e.metaKey || e.ctrlKey || e.altKey) return
      const target = e.target as HTMLElement | null
      if (target?.closest('input, textarea, select, [contenteditable="true"]')) return
      e.preventDefault()
      document.getElementById(SEARCH_INPUT_ID)?.focus()
    }
    window.addEventListener('keydown', focusSearch)
    return () => window.removeEventListener('keydown', focusSearch)
  }, [])

  return (
    <div style={{ minHeight: '100vh', position: 'relative', overflowX: 'hidden' }}>
      {f.view === 'home' ? (
        <HomeHero
          mode={f.searchMode}
          modeHint={f.modeHint}
          language={f.language}
          onSelectMode={f.setMode}
          onSelectLanguage={f.setLanguage}
          onSubmit={(q) => f.submitQuery(q, false)}
        />
      ) : f.view === 'research' ? (
        f.research && (
          <ResearchView
            research={f.research}
            sources={f.researchSources}
            openSteps={f.openSteps}
            searchMode={f.searchMode}
            savedCount={f.savedList.length}
            onToggleStep={f.toggleStep}
            onSelectMode={f.setMode}
            onSubmit={(q) => f.submitQuery(q, false)}
            onHome={f.goHome}
            onToggleBoard={f.toggleBoard}
            onToggleSave={f.toggleSave}
            onFollowup={(q) => f.startResearch(q)}
          />
        )
      ) : (
        <>
          <ResultsHeader
            query={activeQuery}
            mode={f.searchMode}
            savedCount={f.savedList.length}
            onSelectMode={f.setMode}
            onSubmit={(q) => f.submitQuery(q, false)}
            onHome={f.goHome}
            onToggleBoard={f.toggleBoard}
            answersOn={f.answersOn}
            onToggleAnswers={f.toggleAnswers}
          />
          <div style={{ maxWidth: CONTENT_WIDTH, margin: '0 auto', padding: '36px 28px 120px' }}>
            <FilterRow filters={f.filters} onApply={f.applyFilters} />
            {f.entries.map((entry) =>
              entry.reader ? (
                <PageEntry key={entry.id} id={entry.id} reader={entry.reader} focusHeading={entry.focusHeading} />
              ) : (
                <QueryEntry
                  key={entry.id}
                  entry={entry}
                  language={f.language}
                  showAnswers={f.answersOn}
                  onSynth={() => f.runSynth(entry.id)}
                  onSelectLanguage={f.setLanguage}
                  onSetFacet={(key) => f.selectFacet(entry.id, key)}
                  onPrevGroup={() => f.prevGroup(entry.id)}
                  onNextGroup={() => f.nextGroup(entry.id)}
                  onToggleSave={(s) => f.toggleSave(s, entry.query)}
                  onRead={(url) => f.readPage(url, true)}
                  onFollowup={(q) => f.submitQuery(q, true)}
                  onCorrection={f.submitCorrection}
                />
              )
            )}
          </div>
        </>
      )}

      <BoardDrawer
        open={f.boardOpen}
        savedList={f.savedList}
        onClose={f.closeBoard}
        onRemove={f.toggleSave}
      />
    </div>
  )
}
