'use client'

import type { CSSProperties } from 'react'
import { CONTENT_WIDTH } from '@/lib/layout'
import { ResultsHeader } from './ResultsHeader'
import { Followups } from './Followups'
import { Spark, Bookmark, ChevronDown } from './icons'
import type { SourceVM } from '@/lib/folio'
import type { DeepResearchStep } from '@/lib/types'
import type { SearchMode } from '@/lib/useFolioSearch'

type ResearchStatus = 'planning' | 'researching' | 'synthesizing' | 'ready' | 'error'

interface ResearchData {
  status: ResearchStatus
  query: string
  title: string
  overview: string
  metaTime: string
  metaSteps: number
  metaFacts: number
  steps: DeepResearchStep[]
  followups: string[]
  progress: { done: number; total: number } | null
  error: string
}

interface ResearchViewProps {
  research: ResearchData
  sources: (SourceVM & { saved: boolean })[]
  openSteps: Record<string, boolean>
  searchMode: SearchMode
  savedCount: number
  onToggleStep: (id: string) => void
  onSelectMode: (m: SearchMode) => void
  onSubmit: (q: string) => void
  onHome: () => void
  onToggleBoard: () => void
  onToggleSave: (s: SourceVM) => void
  onFollowup: (q: string) => void
}

function pillStyle(): CSSProperties {
  return {
    display: 'flex',
    alignItems: 'center',
    gap: 9,
    padding: '9px 15px',
    borderRadius: 999,
    background: 'var(--surface)',
    boxShadow: 'var(--raise-sm)',
    color: 'var(--muted)',
  }
}

function ClockIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" strokeWidth="2" strokeLinecap="round">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 8v4l3 2" />
    </svg>
  )
}
function StepsIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M4 6h16M4 12h16M4 18h10" />
    </svg>
  )
}
function FactsIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="var(--accent)">
      <circle cx="12" cy="12" r="4" />
    </svg>
  )
}
function FocusIcon() {
  return (
    <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="3.2" />
      <path d="M12 3v3M12 18v3M3 12h3M18 12h3" />
    </svg>
  )
}

function statusLine(research: ResearchData): string {
  if (research.status === 'synthesizing') return 'Writing your path…'
  if (research.status === 'researching') {
    return research.progress ? `Reading ${research.progress.done}/${research.progress.total} sources…` : 'Searching…'
  }
  return ''
}

export function ResearchView({
  research,
  sources,
  openSteps,
  searchMode,
  savedCount,
  onToggleStep,
  onSelectMode,
  onSubmit,
  onHome,
  onToggleBoard,
  onToggleSave,
  onFollowup,
}: ResearchViewProps) {
  const loading = research.status === 'planning'
  const inProgress = research.status === 'researching' || research.status === 'synthesizing'
  const live = statusLine(research)

  return (
    <div>
      <ResultsHeader
        query={research.query}
        mode={searchMode}
        savedCount={savedCount}
        onSelectMode={onSelectMode}
        onSubmit={onSubmit}
        onHome={onHome}
        onToggleBoard={onToggleBoard}
      />

      <div style={{ maxWidth: CONTENT_WIDTH, margin: '0 auto', padding: '40px 28px 160px' }}>
        <div style={{ animation: 'folio-rise .5s ease both' }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 9,
              font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
              letterSpacing: '.14em',
              textTransform: 'uppercase',
              color: 'var(--accent)',
              marginBottom: 12,
            }}
          >
            <Spark size={13} fill="currentColor" />
            Deep research
          </div>

          {loading ? (
            <>
              <div className="folio-skel" style={{ height: 42, width: '55%', marginBottom: 16, borderRadius: 10 }} />
              <div className="folio-skel" style={{ height: 21, width: '85%', marginBottom: 8, borderRadius: 8 }} />
              <div className="folio-skel" style={{ height: 21, width: '70%', marginBottom: 24, borderRadius: 8 }} />
            </>
          ) : research.status === 'error' ? (
            <div
              style={{
                marginBottom: 8,
                padding: '18px 22px',
                borderRadius: 16,
                background: 'var(--surface)',
                boxShadow: 'var(--raise-sm)',
                fontSize: 'var(--fs-body)',

                maxWidth: 720,
              }}
            >
              {research.error || 'Deep research failed.'}
            </div>
          ) : (
            <>
              <h1
                style={{
                  fontFamily: "'Newsreader', serif",
                  fontWeight: 500,
                  fontSize: 42,
                  lineHeight: 1.1,
                  letterSpacing: '-0.02em',
                  margin: '0 0 16px',
                }}
              >
                {research.title}
              </h1>
              <p
                style={{
                  fontFamily: "'Newsreader', serif",
                  fontSize: 'var(--fs-title)',
                  lineHeight: 1.56,
                  maxWidth: 720,
                  margin: '0 0 24px',
                  opacity: 0.9,
                }}
              >
                {research.overview}
              </p>

              <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 11, marginBottom: 36 }}>
                <div style={pillStyle()}>
                  <ClockIcon />
                  <span style={{ font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace" }}>{research.metaTime || '…'}</span>
                </div>
                <div style={pillStyle()}>
                  <StepsIcon />
                  <span style={{ font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace" }}>{research.metaSteps} steps</span>
                </div>
                <div style={pillStyle()}>
                  <FactsIcon />
                  <span style={{ font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace" }}>
                    {research.status === 'ready' ? `${research.metaFacts} atomic facts` : '…'}
                  </span>
                </div>
                {live && (
                  <span
                    style={{
                      font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", color: 'var(--muted)',
                      letterSpacing: '.04em',

                    }}
                  >
                    {live}
                  </span>
                )}
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0,1fr) 316px', gap: 36, alignItems: 'start' }}>
                <div>
                  <div
                    style={{
                      padding: '22px 24px',
                      borderRadius: 20,
                      background: 'var(--surface)',
                      boxShadow: 'var(--raise)',
                      marginBottom: 30,
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: 11, marginBottom: 11 }}>
                      <div
                        style={{
                          display: 'grid',
                          placeItems: 'center',
                          width: 34,
                          height: 34,
                          borderRadius: '50%',
                          background: 'var(--accent-soft)',
                          flexShrink: 0,
                        }}
                      >
                        <FocusIcon />
                      </div>
                      <div style={{ fontFamily: "'Newsreader', serif", fontWeight: 500, fontSize: 'var(--fs-lead)' }}>Built for focus</div>
                    </div>
                    <p style={{ fontSize: 'var(--fs-body)', lineHeight: 1.6, margin: '0 0 16px' }}>
                      This topic is broken into a short, ordered path — open one step at a time. Inside each, the
                      ideas are split into <strong style={{ fontWeight: 600, color: 'var(--text)' }}>atomic facts</strong>: a
                      single thought per card, so nothing competes for your attention. Skim the whole path first to
                      see the shape, then go deep.
                    </p>
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 9 }}>
                      {['One step at a time', 'One idea per card', 'Skim, then dive'].map((tag) => (
                        <span
                          key={tag}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 7,
                            padding: '7px 13px',
                            borderRadius: 999,
                            background: 'var(--bg)',
                            boxShadow: 'var(--inset)',
                            font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", color: 'var(--muted)',
                            letterSpacing: '.03em',
                          }}
                        >
                          <span style={{ width: 5, height: 5, borderRadius: '50%', background: 'var(--accent)' }} />
                          {tag}
                        </span>
                      ))}
                    </div>
                  </div>

                  <div style={{ display: 'flex', alignItems: 'baseline', gap: 10, marginBottom: 16 }}>
                    <span
                      style={{
                        font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", color: 'var(--muted)',
                        letterSpacing: '.14em',
                        textTransform: 'uppercase',

                      }}
                    >
                      The learning path
                    </span>
                    <span style={{ font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace" }}>
                      {research.metaSteps} steps
                    </span>
                  </div>

                  <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
                    {research.steps.map((step) => {
                      const open = !!openSteps[step.id]
                      const ready = research.status === 'ready'
                      return (
                        <div
                          key={step.id}
                          style={{
                            borderRadius: 20,
                            background: 'var(--surface)',
                            boxShadow: open ? 'var(--raise-hi)' : 'var(--raise)',
                            transition: 'box-shadow .35s ease',
                            overflow: 'hidden',
                          }}
                        >
                          <button
                            onClick={() => onToggleStep(step.id)}
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 16,
                              width: '100%',
                              textAlign: 'left',
                              padding: '18px 20px',
                              cursor: 'pointer',
                              background: 'transparent',
                            }}
                          >
                            <span
                              style={{
                                display: 'grid',
                                placeItems: 'center',
                                width: 40,
                                height: 40,
                                borderRadius: '50%',
                                flexShrink: 0,
                                fontFamily: "'IBM Plex Mono', monospace",
                                fontWeight: 500,
                                fontSize: 'var(--fs-body)',
                                background: 'var(--bg)',
                                boxShadow: open ? 'var(--inset)' : 'var(--raise-sm)',
                                color: open ? 'var(--accent)' : 'var(--muted)',
                              }}
                            >
                              {step.n}
                            </span>
                            <span style={{ flex: 1, minWidth: 0 }}>
                              <span style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                                <span style={{ fontFamily: "'Newsreader', serif", fontWeight: 500, fontSize: 'var(--fs-lead)', lineHeight: 1.25 }}>
                                  {step.title}
                                </span>
                                {step.isStart && (
                                  <span
                                    style={{
                                      font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
                                      letterSpacing: '.1em',
                                      textTransform: 'uppercase',
                                      color: '#161616',
                                      background: 'var(--accent)',
                                      padding: '3px 8px',
                                      borderRadius: 999,
                                    }}
                                  >
                                    Start here
                                  </span>
                                )}
                              </span>
                              <span style={{ display: 'block', fontSize: 'var(--fs-body)', marginTop: 3 }}>{step.summary}</span>
                            </span>
                            <span style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: 3, flexShrink: 0 }}>
                              <span style={{ font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", color: 'var(--muted)', letterSpacing: '.04em' }}>
                                {ready ? step.time : '···'}
                              </span>
                              <span style={{ font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", letterSpacing: '.04em', color: 'var(--accent)' }}>
                                {ready ? `${step.facts.length} facts` : '···'}
                              </span>
                            </span>
                            <ChevronDown size={18} />
                          </button>
                          <div
                            style={{
                              maxHeight: open ? 1600 : 0,
                              opacity: open ? 1 : 0,
                              overflow: 'hidden',
                              transition: 'max-height .5s ease, opacity .35s ease',
                            }}
                          >
                            <div style={{ padding: '0 20px 22px 76px' }}>
                              {ready ? (
                                <>
                                  <p style={{ fontSize: 'var(--fs-body)', lineHeight: 1.62, margin: '0 0 20px' }}>{step.desc}</p>
                                  <div
                                    style={{
                                      font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", color: 'var(--muted)',
                                      letterSpacing: '.12em',
                                      textTransform: 'uppercase',

                                      marginBottom: 13,
                                    }}
                                  >
                                    Atomic facts
                                  </div>
                                  <div
                                    style={{
                                      display: 'grid',
                                      gridTemplateColumns: 'repeat(auto-fill, minmax(230px, 1fr))',
                                      gap: 12,
                                    }}
                                  >
                                    {step.facts.map((fact) => (
                                      <div
                                        key={fact.n}
                                        style={{
                                          display: 'flex',
                                          gap: 11,
                                          padding: 15,
                                          borderRadius: 15,
                                          background: 'var(--bg)',
                                          boxShadow: 'var(--inset)',
                                        }}
                                      >
                                        <span
                                          style={{
                                            display: 'grid',
                                            placeItems: 'center',
                                            width: 22,
                                            height: 22,
                                            borderRadius: '50%',
                                            flexShrink: 0,
                                            background: 'var(--accent-soft)',
                                            color: 'var(--accent)',
                                            font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
                                          }}
                                        >
                                          {fact.n}
                                        </span>
                                        <span style={{ fontSize: 'var(--fs-body)', lineHeight: 1.5 }}>{fact.text}</span>
                                      </div>
                                    ))}
                                  </div>
                                </>
                              ) : (
                                <>
                                  <div className="folio-skel" style={{ height: 15, width: '92%', marginBottom: 10 }} />
                                  <div className="folio-skel" style={{ height: 15, width: '70%', marginBottom: 20 }} />
                                  <div className="folio-skel" style={{ height: 60, width: '100%', borderRadius: 15 }} />
                                </>
                              )}
                            </div>
                          </div>
                        </div>
                      )
                    })}
                  </div>

                  {research.status === 'ready' && <Followups items={research.followups} onPick={onFollowup} />}
                </div>

                <div style={{ position: 'sticky', top: 88 }}>
                  <div style={{ display: 'flex', alignItems: 'baseline', gap: 9, marginBottom: 15 }}>
                    <span
                      style={{
                        font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", color: 'var(--muted)',
                        letterSpacing: '.14em',
                        textTransform: 'uppercase',

                      }}
                    >
                      Sources
                    </span>
                    <span style={{ font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace" }}>to go deeper</span>
                  </div>
                  <div
                    style={{
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 12,
                      maxHeight: 'calc(100vh - 130px)',
                      overflowY: 'auto',
                      padding: 2,
                    }}
                  >
                    {sources.length === 0 && inProgress
                      ? [0, 1, 2].map((i) => (
                          <div key={i} className="folio-skel" style={{ height: 110, borderRadius: 16 }} />
                        ))
                      : sources.map((src) => (
                          <div
                            key={src.id}
                            style={{
                              padding: '15px 16px',
                              borderRadius: 16,
                              background: 'var(--surface)',
                              boxShadow: 'var(--raise-sm)',
                            }}
                          >
                            <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 10, marginBottom: 8 }}>
                              <div
                                style={{
                                  display: 'flex',
                                  alignItems: 'center',
                                  gap: 7,
                                  font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", color: 'var(--muted)',
                                  letterSpacing: '.08em',
                                  textTransform: 'uppercase',

                                }}
                              >
                                <span style={{ width: 5, height: 5, borderRadius: '50%', background: 'var(--accent)' }} />
                                {src.type}
                              </div>
                              <button
                                onClick={() => onToggleSave(src)}
                                title="Save to board"
                                style={{
                                  display: 'grid',
                                  placeItems: 'center',
                                  flexShrink: 0,
                                  width: 28,
                                  height: 28,
                                  borderRadius: '50%',
                                  background: src.saved ? 'var(--accent)' : 'transparent',
                                  boxShadow: src.saved ? 'none' : 'var(--raise-sm)',
                                }}
                              >
                                <Bookmark size={13} fill={src.saved ? '#161616' : 'none'} stroke={src.saved ? '#161616' : 'var(--muted)'} />
                              </button>
                            </div>
                            <a
                              href={src.url}
                              target="_blank"
                              rel="noopener noreferrer"
                              style={{ textDecoration: 'none', color: 'inherit', display: 'block' }}
                            >
                              <div style={{ fontFamily: "'Newsreader', serif", fontWeight: 500, fontSize: 'var(--fs-lead)', lineHeight: 1.3, marginBottom: 6 }}>
                                {src.title}
                              </div>
                              <div style={{ font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", color: 'var(--muted)', marginBottom: 8 }}>
                                {src.domain}
                                {src.meta ? ` · ${src.meta}` : ''}
                              </div>
                              <p style={{ fontSize: 'var(--fs-body)', lineHeight: 1.5, margin: 0 }}>{src.snippet}</p>
                            </a>
                          </div>
                        ))}
                  </div>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
