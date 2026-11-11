interface LogoProps {
  size?: number
  onClick?: () => void
}

export function Logo({ size = 23, onClick }: LogoProps) {
  const dot = Math.round(size * 0.43)
  const inner = (
    <>
      <div
        style={{
          width: dot,
          height: dot,
          borderRadius: '50%',
          background: 'var(--accent)',
          boxShadow: `0 0 0 ${Math.round(dot * 0.4)}px var(--accent-soft)`,
        }}
      />
      <span
        style={{
          fontFamily: "'Newsreader', serif",
          fontWeight: 500,
          fontSize: size,
          letterSpacing: '-0.02em',
        }}
      >
        Kurai
      </span>
    </>
  )

  if (onClick) {
    return (
      <button
        onClick={onClick}
        style={{ display: 'flex', alignItems: 'center', gap: 10, flexShrink: 0 }}
        aria-label="Kurai home"
      >
        {inner}
      </button>
    )
  }

  return <div style={{ display: 'flex', alignItems: 'center', gap: 13 }}>{inner}</div>
}
