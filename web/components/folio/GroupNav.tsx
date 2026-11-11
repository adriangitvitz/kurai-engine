'use client'

import type { CSSProperties } from 'react'

interface GroupNavProps {
  rangeStart: number
  rangeEnd: number
  total: number
  groupCount: number
  hasMore: boolean
  canPrev: boolean
  canNext: boolean
  loadingMore: boolean
  onPrev: () => void
  onNext: () => void
}

function btnStyle(enabled: boolean): CSSProperties {
  return {
    display: 'grid',
    placeItems: 'center',
    width: 36,
    height: 36,
    borderRadius: '50%',
    background: 'var(--surface)',
    color: enabled ? 'var(--accent)' : 'var(--muted)',
    boxShadow: enabled ? 'var(--raise-sm)' : 'var(--inset)',
    cursor: enabled ? 'pointer' : 'not-allowed',
    transition: 'box-shadow .2s, color .2s',
  }
}

export function GroupNav({
  rangeStart,
  rangeEnd,
  total,
  groupCount,
  hasMore,
  canPrev,
  canNext,
  loadingMore,
  onPrev,
  onNext,
}: GroupNavProps) {
  if (groupCount <= 1 && !hasMore) return null

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
      <span style={{ font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", letterSpacing: '.04em', color: 'var(--muted)' }}>
        {rangeStart}–{rangeEnd} of {Math.max(total, rangeEnd)}
      </span>
      <button onClick={onPrev} disabled={!canPrev} aria-label="Previous results" style={btnStyle(canPrev)}>
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round">
          <path d="M15 6l-6 6 6 6" />
        </svg>
      </button>
      <button onClick={onNext} disabled={!canNext || loadingMore} aria-label="Next results" style={btnStyle(canNext && !loadingMore)}>
        {loadingMore ? (
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" style={{ animation: 'folio-spin .8s linear infinite' }}>
            <path d="M21 12a9 9 0 1 1-6.2-8.5" />
          </svg>
        ) : (
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round">
            <path d="M9 6l6 6-6 6" />
          </svg>
        )}
      </button>
    </div>
  )
}
