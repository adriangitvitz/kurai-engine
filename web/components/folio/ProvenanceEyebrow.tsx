'use client'

interface ProvenanceEyebrowProps {
  label: string
  source: string
  sourceHref?: string
}

const eyebrowFont = {
  font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
  letterSpacing: '.08em',
  textTransform: 'uppercase' as const,
  color: 'var(--muted)',
}

export function ProvenanceEyebrow({ label, source, sourceHref }: ProvenanceEyebrowProps) {
  return (
    <div style={{ ...eyebrowFont, marginBottom: 12, display: 'flex', gap: 6, alignItems: 'baseline' }}>
      <span>{label}</span>
      <span aria-hidden="true">·</span>
      {sourceHref ? (
        <a
          href={sourceHref}
          target="_blank"
          rel="noopener noreferrer"
          style={{ ...eyebrowFont, color: 'var(--accent)', textDecoration: 'underline', textUnderlineOffset: 3 }}
        >
          {source}
        </a>
      ) : (
        <span>{source}</span>
      )}
    </div>
  )
}
