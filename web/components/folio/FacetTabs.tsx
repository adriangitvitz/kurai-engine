import type { CSSProperties } from 'react'
import type { Facet } from '@/lib/folio'

interface FacetTabsProps {
  facets: Facet[]
  active: string
  onSelect: (key: string) => void
}

function tabStyle(active: boolean): CSSProperties {
  return {
    display: 'flex',
    alignItems: 'center',
    gap: 7,
    padding: '8px 14px',
    borderRadius: 999,
    fontWeight: 600,
    fontSize: 'var(--fs-body)',
    cursor: 'pointer',
    whiteSpace: 'nowrap',
    transition: 'all .2s',
    background: 'var(--surface)',
    boxShadow: active ? 'var(--inset)' : 'var(--raise-sm)',
    color: active ? 'var(--accent)' : 'var(--muted)',
  }
}

export function FacetTabs({ facets, active, onSelect }: FacetTabsProps) {
  if (facets.length <= 1) return null
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 9, marginBottom: 22 }}>
      {facets.map((f) => (
        <button key={f.key} onClick={() => onSelect(f.key)} style={tabStyle(active === f.key)}>
          {f.label}
          <span style={{ fontFamily: "'IBM Plex Mono', monospace", fontSize: 'var(--fs-body)' }}>
            {f.count}
          </span>
        </button>
      ))}
    </div>
  )
}
