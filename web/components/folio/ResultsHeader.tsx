'use client'

import { useState } from 'react'
import { Logo } from './Logo'
import { ModeToggle } from './ModeToggle'
import { AnswersToggle } from './AnswersToggle'
import { SearchIcon, ArrowRight, Bookmark } from './icons'
import type { SearchMode } from '@/lib/useFolioSearch'
import { CONTENT_WIDTH } from '@/lib/layout'
import { SEARCH_INPUT_ID } from '@/lib/nav'

interface ResultsHeaderProps {
  query: string
  mode: SearchMode
  savedCount: number
  onSelectMode: (m: SearchMode) => void
  onSubmit: (q: string) => void
  onHome: () => void
  onToggleBoard: () => void
  answersOn?: boolean
  onToggleAnswers?: () => void
  maxWidth?: number
}

export function ResultsHeader({
  query,
  mode,
  savedCount,
  onSelectMode,
  onSubmit,
  onHome,
  onToggleBoard,
  answersOn = true,
  onToggleAnswers,
  maxWidth = CONTENT_WIDTH,
}: ResultsHeaderProps) {
  const [input, setInput] = useState(query)
  const [prevQuery, setPrevQuery] = useState(query)
  if (query !== prevQuery) {
    setPrevQuery(query)
    setInput(query)
  }

  return (
    <div
      style={{
        position: 'sticky',
        top: 0,
        zIndex: 30,
        backdropFilter: 'blur(8px)',
        background: 'var(--bg-trans)',
        borderBottom: '1px solid var(--line)',
      }}
    >
      <div
        style={{
          maxWidth,
          margin: '0 auto',
          display: 'flex',
          alignItems: 'center',
          gap: 18,
          padding: '13px 28px',
        }}
      >
        <Logo size={23} onClick={onHome} />

        <form
          role="search"
          action="/"
          method="get"
          onSubmit={(e) => {
            e.preventDefault()
            onSubmit(input)
          }}
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            padding: '7px 8px 7px 18px',
            borderRadius: 999,
            background: 'var(--bg)',
            boxShadow: 'var(--inset)',
            maxWidth: 440,
          }}
        >
          <SearchIcon size={16} style={{ }} />
          <input
            type="search"
            id={SEARCH_INPUT_ID}
            name="q"
            aria-label="Search"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            style={{
              flex: 1,
              border: 'none',
              outline: 'none',
              background: 'transparent',
              fontSize: 'var(--fs-body)',
              color: 'var(--text)',
            }}
          />
          <button
            type="submit"
            aria-label="Search"
            style={{
              display: 'grid',
              placeItems: 'center',
              width: 34,
              height: 34,
              borderRadius: '50%',
              background: 'var(--accent)',
            }}
          >
            <ArrowRight size={15} />
          </button>
        </form>

        <ModeToggle mode={mode} onSelect={onSelectMode} mini />

        {onToggleAnswers && <AnswersToggle on={answersOn} onToggle={onToggleAnswers} />}

        <button
          onClick={onToggleBoard}
          aria-label={savedCount > 0 ? `Board, ${savedCount} saved` : 'Board'}
          style={{
            position: 'relative',
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            padding: '0 15px',
            height: 40,
            borderRadius: 999,
            background: 'var(--surface)',
            boxShadow: 'var(--raise-sm)',
            fontSize: 'var(--fs-body)',
            fontWeight: 600,
            flexShrink: 0,
          }}
        >
          <Bookmark size={16} stroke="currentColor" />
          {savedCount > 0 && (
            <span
              style={{
                minWidth: 19,
                height: 19,
                padding: '0 5px',
                borderRadius: 999,
                background: 'var(--accent)',
                color: '#161616',
                font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
                display: 'grid',
                placeItems: 'center',
              }}
            >
              {savedCount}
            </span>
          )}
        </button>
      </div>
    </div>
  )
}
