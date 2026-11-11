import type { CSSProperties } from 'react'
import type { Language } from '@/lib/useFolioSearch'

interface LangToggleProps {
  language: Language
  onSelect: (l: Language) => void
  mini?: boolean
}

const OPTIONS: { key: Language; label: string }[] = [
  { key: 'en', label: 'EN' },
  { key: 'es', label: 'ES' },
]

function segStyle(active: boolean, mini: boolean): CSSProperties {
  return {
    padding: mini ? '5px 11px' : '7px 14px',
    borderRadius: 999,
    fontSize: 'var(--fs-meta)',
    fontWeight: 600,
    letterSpacing: '.04em',
    whiteSpace: 'nowrap',
    transition: 'all .22s ease',
    background: active ? 'var(--surface)' : 'transparent',
    boxShadow: active ? 'var(--raise-sm)' : 'none',
    color: active ? 'var(--accent)' : 'var(--muted)',
  }
}

export function LangToggle({ language, onSelect, mini = false }: LangToggleProps) {
  return (
    <div
      style={{
        display: 'inline-flex',
        gap: 4,
        padding: mini ? 4 : 5,
        borderRadius: 999,
        background: 'var(--bg)',
        boxShadow: 'var(--inset)',
        flexShrink: 0,
      }}
      title="Answer language"
    >
      {OPTIONS.map((o) => (
        <button key={o.key} onClick={() => onSelect(o.key)} style={segStyle(language === o.key, mini)}>
          {o.label}
        </button>
      ))}
    </div>
  )
}
