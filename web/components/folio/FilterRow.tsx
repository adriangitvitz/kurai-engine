'use client'

import { useState, type CSSProperties } from 'react'
import { SearchFilters, NO_FILTERS, parseSites, hasFilters } from '@/lib/filters'

interface FilterRowProps {
  filters: SearchFilters
  onApply: (f: SearchFilters) => void
}

const LABEL: CSSProperties = {
  font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
  letterSpacing: '.08em',
  textTransform: 'uppercase',
  color: 'var(--muted)',
}

const FIELD: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 10,
  padding: '8px 16px',
  borderRadius: 999,
  background: 'var(--bg)',
  boxShadow: 'var(--inset)',
}

const INPUT: CSSProperties = {
  border: 'none',
  outline: 'none',
  background: 'transparent',
  fontSize: 'var(--fs-body)',
  color: 'var(--text)',
  colorScheme: 'dark',
}

export function FilterRow({ filters, onApply }: FilterRowProps) {
  const [sites, setSites] = useState(filters.sites.join(', '))
  const [prev, setPrev] = useState(filters)
  if (filters !== prev) {
    setPrev(filters)
    setSites(filters.sites.join(', '))
  }

  return (
    <form
      aria-label="Filters"
      onSubmit={(e) => {
        e.preventDefault()
        onApply({ ...filters, sites: parseSites(sites) })
      }}
      style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 12, marginBottom: 32 }}
    >
      <label style={FIELD}>
        <span style={LABEL}>Site</span>
        <input
          id="folio-filter-site"
          name="site"
          value={sites}
          onChange={(e) => setSites(e.target.value)}
          onBlur={() => {
            const next = parseSites(sites)
            if (next.join(',') !== filters.sites.join(',')) onApply({ ...filters, sites: next })
          }}
          placeholder="example.com"
          autoComplete="off"
          spellCheck={false}
          style={{ ...INPUT, width: 220 }}
        />
      </label>
      <label style={FIELD}>
        <span style={LABEL}>From</span>
        <input
          id="folio-filter-from"
          name="from"
          type="date"
          value={filters.from}
          max={filters.to || undefined}
          onChange={(e) => onApply({ ...filters, from: e.target.value })}
          style={INPUT}
        />
      </label>
      <label style={FIELD}>
        <span style={LABEL}>To</span>
        <input
          id="folio-filter-to"
          name="to"
          type="date"
          value={filters.to}
          min={filters.from || undefined}
          onChange={(e) => onApply({ ...filters, to: e.target.value })}
          style={INPUT}
        />
      </label>
      {hasFilters(filters) && (
        <button
          type="button"
          onClick={() => onApply(NO_FILTERS)}
          style={{
            padding: '8px 14px',
            borderRadius: 999,
            background: 'var(--surface)',
            boxShadow: 'var(--raise-sm)',
            fontSize: 'var(--fs-body)',
            fontWeight: 600,
            color: 'var(--accent)',
          }}
        >
          Clear filters
        </button>
      )}
    </form>
  )
}
