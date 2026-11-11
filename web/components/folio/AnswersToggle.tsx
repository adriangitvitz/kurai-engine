'use client'

import { Cards } from './icons'

interface AnswersToggleProps {
  on: boolean
  onToggle: () => void
}

export function AnswersToggle({ on, onToggle }: AnswersToggleProps) {
  const label = 'Answer cards & panels'
  return (
    <button
      role="switch"
      aria-checked={on}
      aria-label={label}
      title={`${label}: ${on ? 'shown' : 'hidden'}`}
      onClick={onToggle}
      style={{
        height: 44,
        padding: 2,
        background: 'transparent',
        border: 'none',
        cursor: 'pointer',
        flexShrink: 0,
        display: 'grid',
        placeItems: 'center',
      }}
    >
      <span
        style={{
          display: 'grid',
          placeItems: 'center',
          width: 40,
          height: 40,
          borderRadius: 999,
          background: on ? 'var(--surface)' : 'var(--bg)',
          boxShadow: on ? 'var(--raise-sm)' : 'var(--inset)',
          color: on ? 'var(--accent)' : 'var(--muted)',
        }}
      >
        <Cards size={17} slashed={!on} />
      </span>
    </button>
  )
}
