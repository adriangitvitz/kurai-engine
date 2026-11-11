'use client'

interface CorrectionsLineProps {
  corrections: string[]
  onPick: (query: string) => void
}

export function CorrectionsLine({ corrections, onPick }: CorrectionsLineProps) {
  const items = corrections.slice(0, 3)
  if (items.length === 0) return null

  return (
    <div
      role="group"
      aria-label="Did you mean"
      style={{
        display: 'flex',
        alignItems: 'baseline',
        flexWrap: 'wrap',
        gap: 8,
        padding: '4px 0',
        marginBottom: 20,
      }}
    >
      <span
        style={{
          font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
          letterSpacing: '.08em',
          textTransform: 'uppercase',
          color: 'var(--muted)',
        }}
      >
        Did you mean
      </span>
      {items.map((c, i) => (
        <span key={c} style={{ display: 'inline-flex', alignItems: 'baseline', gap: 8 }}>
          {i > 0 && (
            <span aria-hidden="true" style={{ font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", color: 'var(--muted)' }}>
              ·
            </span>
          )}
          <button
            onClick={() => onPick(c)}
            style={{
              background: 'none',
              border: 'none',
              padding: 0,
              cursor: 'pointer',
              font: "var(--fw-body) var(--fs-body) 'Hanken Grotesk', sans-serif",
              color: 'var(--accent)',
              textDecoration: 'underline',
              textUnderlineOffset: 3,
            }}
          >
            {c}
          </button>
        </span>
      ))}
    </div>
  )
}
