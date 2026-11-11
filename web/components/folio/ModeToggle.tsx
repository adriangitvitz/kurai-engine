import type { CSSProperties } from 'react'
import { Spark, BookOpen } from './icons'
import type { SearchMode } from '@/lib/useFolioSearch'

interface ModeToggleProps {
  mode: SearchMode
  onSelect: (m: SearchMode) => void
  mini?: boolean
}

function segStyle(active: boolean, mini: boolean): CSSProperties {
  return {
    display: 'flex',
    alignItems: 'center',
    gap: 6,
    padding: mini ? '7px 13px' : '8px 16px',
    borderRadius: 999,
    fontSize: 'var(--fs-meta)',
    fontWeight: 600,
    whiteSpace: 'nowrap',
    transition: 'all .22s ease',
    background: active ? 'var(--surface)' : 'transparent',
    boxShadow: active ? 'var(--raise-sm)' : 'none',
    color: active ? 'var(--accent)' : 'var(--muted)',
  }
}

export function ModeToggle({ mode, onSelect, mini = false }: ModeToggleProps) {
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
    >
      <button onClick={() => onSelect('sources')} style={segStyle(mode === 'sources', mini)}>
        {mini ? 'Sources' : 'Sources only'}
      </button>
      <button onClick={() => onSelect('synth')} style={segStyle(mode === 'synth', mini)}>
        <Spark size={mini ? 11 : 12} />
        {mini ? 'Answer' : 'Synthesized answer'}
      </button>
      <button onClick={() => onSelect('deep')} style={segStyle(mode === 'deep', mini)}>
        <BookOpen size={mini ? 11 : 12} />
        {mini ? 'Learn' : 'Deep research'}
      </button>
    </div>
  )
}
