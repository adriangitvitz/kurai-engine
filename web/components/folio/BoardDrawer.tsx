'use client'

import { useMemo, useState, type CSSProperties } from 'react'
import { Close, Bookmark } from './icons'
import type { SourceVM } from '@/lib/folio'

interface BoardDrawerProps {
  open: boolean
  savedList: SourceVM[]
  onClose: () => void
  onRemove: (s: SourceVM) => void
}

type BoardView = 'flat' | 'grouped'

const metaFont: CSSProperties = {
  font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
  letterSpacing: '.03em',
  color: 'var(--muted)',
}

function savedAtLabel(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric' }).format(d)
}

function BoardItem({
  item,
  onRemove,
  showProvenance = true,
}: {
  item: SourceVM
  onRemove: (s: SourceVM) => void
  showProvenance?: boolean
}) {
  const saved = savedAtLabel(item.savedAt)
  return (
    <div
      style={{
        display: 'flex',
        gap: 13,
        alignItems: 'flex-start',
        padding: '15px 16px',
        borderRadius: 16,
        background: 'var(--surface)',
        boxShadow: 'var(--raise-sm)',
      }}
    >
      <div style={{ width: 6, height: 6, borderRadius: '50%', background: 'var(--accent)', marginTop: 7, flexShrink: 0 }} />
      <a
        href={item.url}
        target="_blank"
        rel="noopener noreferrer"
        style={{ flex: 1, minWidth: 0, textDecoration: 'none', color: 'inherit' }}
      >
        <div style={{ fontFamily: "'Newsreader', serif", fontWeight: 500, fontSize: 'var(--fs-lead)', lineHeight: 1.3, marginBottom: 4 }}>
          {item.title}
        </div>
        <div style={metaFont}>
          {item.domain}
          {item.type ? ` · ${item.type}` : ''}
          {saved ? ` · saved ${saved}` : ''}
        </div>
        {showProvenance && item.query && (
          <div style={{ ...metaFont, marginTop: 5, textTransform: 'uppercase', letterSpacing: '.06em' }}>
            Saved while searching “{item.query}”
          </div>
        )}
      </a>
      <button onClick={() => onRemove(item)} title="Remove" aria-label={`Remove ${item.title}`} style={{ color: 'var(--muted)', padding: 3 }}>
        <Close size={14} />
      </button>
    </div>
  )
}

export function BoardDrawer({ open, savedList, onClose, onRemove }: BoardDrawerProps) {
  const [view, setView] = useState<BoardView>('flat')
  const [mounted, setMounted] = useState(open)
  if (open && !mounted) setMounted(true)

  const groups = useMemo(() => {
    const out: { query?: string; items: SourceVM[] }[] = []
    const index = new Map<string, number>()
    for (const s of savedList) {
      const key = s.query ?? ''
      const at = index.get(key)
      if (at === undefined) {
        out.push({ query: s.query, items: [s] })
        index.set(key, out.length - 1)
      } else {
        out[at].items.push(s)
      }
    }
    return out
  }, [savedList])

  const scrimStyle: CSSProperties = {
    position: 'fixed',
    inset: 0,
    zIndex: 40,
    background: 'rgba(0,0,0,0.45)',
    opacity: open ? 1 : 0,
    pointerEvents: open ? 'auto' : 'none',
    transition: 'opacity .35s ease',
  }
  const drawerStyle: CSSProperties = {
    position: 'fixed',
    top: 0,
    right: 0,
    bottom: 0,
    zIndex: 50,
    width: 380,
    maxWidth: '88vw',
    display: 'flex',
    flexDirection: 'column',
    background: 'var(--bg)',
    boxShadow: '-14px 0 40px rgba(0,0,0,0.5)',
    transform: open ? 'translateX(0)' : 'translateX(102%)',
    visibility: open ? 'visible' : 'hidden',
    transition: `transform .42s cubic-bezier(.4,0,.2,1), visibility 0s linear ${open ? '0s' : '.42s'}`,
  }

  const tabStyle = (active: boolean): CSSProperties => ({
    padding: '6px 13px',
    borderRadius: 999,
    font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
    letterSpacing: '.06em',
    textTransform: 'uppercase',
    background: active ? 'var(--bg)' : 'transparent',
    boxShadow: active ? 'var(--inset)' : 'none',
    color: active ? 'var(--accent)' : 'var(--muted)',
  })

  return (
    <>
      <div style={scrimStyle} onClick={onClose} />
      <div
        style={drawerStyle}
        inert={!open}
        onTransitionEnd={(e) => {
          if (!open && e.target === e.currentTarget && e.propertyName === 'transform') setMounted(false)
        }}
      >
        {mounted && (
          <>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '26px 26px 18px' }}>
              <div>
                <div
                  style={{
                    font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
                    letterSpacing: '.14em',
                    textTransform: 'uppercase',
                    color: 'var(--accent)',
                    marginBottom: 7,
                  }}
                >
                  Board
                </div>
                <div style={{ fontFamily: "'Newsreader', serif", fontWeight: 500, fontSize: 26 }}>My research</div>
              </div>
              <button
                onClick={onClose}
                aria-label="Close board"
                style={{
                  display: 'grid',
                  placeItems: 'center',
                  width: 38,
                  height: 38,
                  borderRadius: '50%',
                  background: 'var(--surface)',
                  boxShadow: 'var(--raise-sm)',
                }}
              >
                <Close size={16} />
              </button>
            </div>

            {savedList.length > 0 && (
              <div role="group" aria-label="Board view" style={{ display: 'flex', gap: 4, padding: '0 26px 16px' }}>
                <button aria-pressed={view === 'flat'} onClick={() => setView('flat')} style={tabStyle(view === 'flat')}>
                  All
                </button>
                <button aria-pressed={view === 'grouped'} onClick={() => setView('grouped')} style={tabStyle(view === 'grouped')}>
                  By search
                </button>
              </div>
            )}

            <div style={{ padding: '0 26px 30px', overflowY: 'auto', flex: 1 }}>
              {savedList.length === 0 ? (
                <div
                  style={{
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    textAlign: 'center',
                    padding: '60px 20px',

                  }}
                >
                  <div style={{ marginBottom: 16 }}>
                    <Bookmark size={34} stroke="currentColor" />
                  </div>
                  <div style={{ fontSize: 'var(--fs-body)', lineHeight: 1.5, maxWidth: 200 }}>
                    Tap the bookmark on any source to collect it here.
                  </div>
                </div>
              ) : view === 'flat' ? (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 13 }}>
                  {savedList.map((item) => (
                    <BoardItem key={item.id} item={item} onRemove={onRemove} />
                  ))}
                </div>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 26 }}>
                  {groups.map((g) => (
                    <section key={g.query ?? '__none__'} aria-label={g.query ? `Saved while searching ${g.query}` : 'Saved without a recorded search'}>
                      <div
                        style={{
                          font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
                          letterSpacing: '.08em',
                          textTransform: 'uppercase',
                          color: 'var(--muted)',
                          marginBottom: 11,
                        }}
                      >
                        {g.query ? `“${g.query}” · ${g.items.length}` : `No search recorded · ${g.items.length}`}
                      </div>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: 13 }}>
                        {g.items.map((item) => (
                          <BoardItem key={item.id} item={item} onRemove={onRemove} showProvenance={false} />
                        ))}
                      </div>
                    </section>
                  ))}
                </div>
              )}
            </div>
          </>
        )}
      </div>
    </>
  )
}
