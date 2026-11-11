'use client'

import { useEffect, useRef } from 'react'
import { AnswerZone } from './AnswerZone'
import { ConversionCard } from './ConversionCard'
import { CorrectionsLine } from './CorrectionsLine'
import { KnowledgePanel } from './KnowledgePanel'
import { WeatherCard } from './WeatherCard'
import { FacetTabs } from './FacetTabs'
import { SourceCard } from './SourceCard'
import { GroupNav } from './GroupNav'
import { Followups } from './Followups'
import type { SourceVM } from '@/lib/folio'
import type { EntryView, Language } from '@/lib/useFolioSearch'

interface QueryEntryProps {
  entry: EntryView
  language: Language
  showAnswers: boolean
  onSynth: () => void
  onSelectLanguage: (l: Language) => void
  onSetFacet: (key: string) => void
  onPrevGroup: () => void
  onNextGroup: () => void
  onToggleSave: (s: SourceVM) => void
  onRead: (url: string) => void
  onFollowup: (q: string) => void
  onCorrection: (q: string) => void
}

export function QueryEntry({
  entry,
  language,
  showAnswers,
  onSynth,
  onSelectLanguage,
  onSetFacet,
  onPrevGroup,
  onNextGroup,
  onToggleSave,
  onRead,
  onFollowup,
  onCorrection,
}: QueryEntryProps) {
  const headingRef = useRef<HTMLHeadingElement>(null)
  const focusedOnce = useRef(false)
  useEffect(() => {
    if (entry.focusHeading && !focusedOnce.current) {
      focusedOnce.current = true
      headingRef.current?.focus()
    }
  }, [entry.focusHeading])

  const corrections = entry.answers?.corrections ?? []

  return (
    <div data-entry-id={entry.id} style={{ marginBottom: 56, animation: 'folio-rise .5s ease both' }}>
      <div
        style={{
          font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
          letterSpacing: '.14em',
          textTransform: 'uppercase',
          color: 'var(--accent)',
          marginBottom: 10,
        }}
      >
        Question
      </div>
      <h1
        ref={headingRef}
        tabIndex={-1}
        style={{
          fontFamily: "'Newsreader', serif",
          fontWeight: 500,
          fontSize: 34,
          lineHeight: 1.18,
          letterSpacing: '-0.015em',
          margin: '0 0 24px',
          outline: 'none',
        }}
      >
        {entry.query}
      </h1>

      {entry.status === 'loading' && (
        <div>
          <div className="folio-skel" style={{ height: 18, width: '90%', marginBottom: 12 }} />
          <div className="folio-skel" style={{ height: 18, width: '78%', marginBottom: 28 }} />
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(6,1fr)', gap: 18 }}>
            <div className="folio-skel" style={{ gridColumn: 'span 6', height: 150, borderRadius: 20 }} />
            <div className="folio-skel" style={{ gridColumn: 'span 3', height: 220, borderRadius: 20 }} />
            <div className="folio-skel" style={{ gridColumn: 'span 3', height: 220, borderRadius: 20 }} />
          </div>
        </div>
      )}

      {entry.status === 'error' && (
        <div
          style={{
            padding: '18px 22px',
            borderRadius: 16,
            background: 'var(--surface)',
            boxShadow: 'var(--raise-sm)',
            fontSize: 'var(--fs-body)',

          }}
        >
          {entry.error || 'Something went wrong.'}
        </div>
      )}

      {entry.status === 'ready' && (
        <>
          {corrections.length > 0 && <CorrectionsLine corrections={corrections} onPick={onCorrection} />}
          {entry.phraseRelaxed && (
            <div role="status" style={{ fontSize: 'var(--fs-body)', color: 'var(--muted)', marginBottom: 20 }}>
              No result contains the exact quoted phrase. Showing approximate matches.
            </div>
          )}

          {showAnswers && entry.answers?.currency && <ConversionCard currency={entry.answers.currency} />}
          {showAnswers && !entry.answers?.currency && entry.answers?.weather && (
            <WeatherCard weather={entry.answers.weather} />
          )}

          <AnswerZone
            synth={entry.synth}
            answer={entry.answer}
            synthError={entry.synthError}
            sourceCount={entry.sourceCount}
            language={language}
            onSynth={onSynth}
            onSelectLanguage={onSelectLanguage}
          />

          {showAnswers && entry.answers?.infobox && (
            <div style={{ marginTop: 28 }}>
              <KnowledgePanel infobox={entry.answers.infobox} />
            </div>
          )}

          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 16,
              margin: '40px 0 16px',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 10 }}>
              <span
                style={{
                  font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", color: 'var(--muted)',
                  letterSpacing: '.14em',
                  textTransform: 'uppercase',

                }}
              >
                Sources &amp; reading
              </span>
              <span style={{ font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace" }}>
                {entry.total} {entry.total === 1 ? 'result' : 'results'}
              </span>
            </div>
            <GroupNav
              rangeStart={entry.rangeStart}
              rangeEnd={entry.rangeStart + entry.cards.length - 1}
              total={entry.total}
              groupCount={entry.groupCount}
              hasMore={entry.hasMore}
              canPrev={entry.canPrev}
              canNext={entry.canNext}
              loadingMore={entry.loadingMore}
              onPrev={onPrevGroup}
              onNext={onNextGroup}
            />
          </div>

          {entry.dateFiltered && (
            <div data-date-notice role="status" style={{ fontSize: 'var(--fs-body)', color: 'var(--muted)', marginBottom: 16 }}>
              Results with a date are filtered to the range. Results without one can’t be checked:{' '}
              {entry.timeRange
                ? `the engines limit them to the past ${entry.timeRange}.`
                : 'they are shown as they came.'}
            </div>
          )}

          <FacetTabs facets={entry.facets} active={entry.facet} onSelect={onSetFacet} />

          {entry.cards.length > 0 ? (
            <div
              key={entry.groupIndex}
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(6,1fr)',
                gap: 18,
                animation: 'folio-rise .35s ease both',
                opacity: entry.loadingMore ? 0.55 : 1,
                transition: 'opacity .2s ease',
              }}
            >
              {entry.cards.map((card) => (
                <SourceCard
                  key={card.source.id}
                  card={card}
                  query={entry.query}
                  onToggleSave={() => onToggleSave(card.source)}
                  onRead={() => onRead(card.source.url)}
                />
              ))}
            </div>
          ) : (
            <div style={{ fontSize: 'var(--fs-body)', padding: '8px 0' }}>No results for this filter.</div>
          )}

          {entry.showFollowups && <Followups items={entry.followupItems} onPick={onFollowup} />}
        </>
      )}
    </div>
  )
}
