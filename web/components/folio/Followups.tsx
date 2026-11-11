interface FollowupsProps {
  items: string[]
  onPick: (q: string) => void
}

export function Followups({ items, onPick }: FollowupsProps) {
  if (items.length === 0) return null
  return (
    <div style={{ marginTop: 34 }}>
      <div
        style={{
          font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
              color: 'var(--muted)',
          letterSpacing: '.14em',
          textTransform: 'uppercase',

          marginBottom: 14,
        }}
      >
        Keep exploring
      </div>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 11 }}>
        {items.map((f) => (
          <button
            key={f}
            onClick={() => onPick(f)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 9,
              padding: '11px 16px',
              borderRadius: 999,
              background: 'var(--surface)',
              boxShadow: 'var(--raise-sm)',
              fontSize: 'var(--fs-body)',
            }}
          >
            <span style={{ color: 'var(--accent)', fontWeight: 700 }}>+</span>
            {f}
          </button>
        ))}
      </div>
    </div>
  )
}
