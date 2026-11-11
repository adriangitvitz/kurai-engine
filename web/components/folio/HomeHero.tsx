'use client'

import { useState } from 'react'
import { ModeToggle } from './ModeToggle'
import { LangToggle } from './LangToggle'
import { SearchIcon, ArrowRight } from './icons'
import type { SearchMode, Language } from '@/lib/useFolioSearch'
import { SEARCH_INPUT_ID } from '@/lib/nav'

interface HomeHeroProps {
  mode: SearchMode
  modeHint: string
  language: Language
  onSelectMode: (m: SearchMode) => void
  onSelectLanguage: (l: Language) => void
  onSubmit: (q: string) => void
}

export function HomeHero({
  mode,
  modeHint,
  language,
  onSelectMode,
  onSelectLanguage,
  onSubmit,
}: HomeHeroProps) {
  const [input, setInput] = useState('')

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '40px 24px 120px',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 13, marginBottom: 18 }}>
        <div
          style={{
            width: 13,
            height: 13,
            borderRadius: '50%',
            background: 'var(--accent)',
            boxShadow: '0 0 0 5px var(--accent-soft)',
          }}
        />
        <div style={{ fontFamily: "'Newsreader', serif", fontWeight: 500, fontSize: 46, letterSpacing: '-0.02em' }}>
          Kurai
        </div>
      </div>
      <div
        style={{
          font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
              color: 'var(--muted)',
          letterSpacing: '.14em',
          textTransform: 'uppercase',

          marginBottom: 42,
        }}
      >
        A calmer way to search the web
      </div>

      <div style={{ width: '100%', maxWidth: 620 }}>
        <form
          role="search"
          action="/"
          method="get"
          onSubmit={(e) => {
            e.preventDefault()
            onSubmit(input)
          }}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            padding: '9px 9px 9px 22px',
            borderRadius: 999,
            background: 'var(--bg)',
            boxShadow: 'var(--inset)',
          }}
        >
          <SearchIcon size={19} style={{ flexShrink: 0 }} />
          <input
            type="search"
            id={SEARCH_INPUT_ID}
            name="q"
            aria-label="Search"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Ask anything, or paste a link"
            autoFocus
            style={{
              flex: 1,
              border: 'none',
              outline: 'none',
              background: 'transparent',
              fontSize: 'var(--fs-lead)',
              color: 'var(--text)',
              padding: '8px 0',
            }}
          />
          <button
            type="submit"
            aria-label="Search"
            style={{
              display: 'grid',
              placeItems: 'center',
              width: 48,
              height: 48,
              borderRadius: '50%',
              background: 'var(--accent)',
              flexShrink: 0,
            }}
          >
            <ArrowRight size={20} />
          </button>
        </form>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 9, marginTop: 24 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <ModeToggle mode={mode} onSelect={onSelectMode} />
          {(mode === 'synth' || mode === 'deep') && <LangToggle language={language} onSelect={onSelectLanguage} />}
        </div>
        <div
          style={{
            font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
              color: 'var(--muted)',
            letterSpacing: '.06em',
            textTransform: 'uppercase',

          }}
        >
          {modeHint}
        </div>
      </div>
    </div>
  )
}
