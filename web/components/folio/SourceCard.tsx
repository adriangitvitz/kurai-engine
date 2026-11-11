'use client'

import { useState, type CSSProperties } from 'react'
import { Bookmark, ArrowRight } from './icons'
import { extractPassages } from '@/lib/api'
import type { Passage } from '@/lib/types'
import type { CardVM } from '@/lib/useFolioSearch'

interface SourceCardProps {
  card: CardVM
  query: string
  onToggleSave: () => void
  onRead: () => void
}

function pointText(p: Passage): string {
  if (p.heading && p.heading.trim()) return p.heading.trim()
  const body = (p.text ?? '').trim()
  if (!body) return ''
  const parts = body.split(/\s+/)
  return parts.slice(0, 14).join(' ') + (parts.length > 14 ? '…' : '')
}

export function SourceCard({ card, query, onToggleSave, onRead }: SourceCardProps) {
  const { source, size, hasImg, saved } = card
  const [hovered, setHovered] = useState(false)
  const [points, setPoints] = useState<string[] | null>(null)
  const [loadingPoints, setLoadingPoints] = useState(false)

  const feature = size === 'feature'
  const span = feature ? 'span 6' : size === 'compact' ? 'span 2' : 'span 3'

  const onEnter = () => {
    setHovered(true)
    if (points === null && !loadingPoints) {
      setLoadingPoints(true)
      extractPassages(source.url, query, 3)
        .then((ps) => setPoints(ps.map(pointText).filter(Boolean).slice(0, 3)))
        .catch(() => setPoints([]))
        .finally(() => setLoadingPoints(false))
    }
  }

  const cardStyle: CSSProperties = {
    gridColumn: span,
    position: 'relative',
    display: 'flex',
    flexDirection: feature ? 'row' : 'column',
    background: 'var(--surface)',
    borderRadius: 20,
    overflow: 'hidden',
    boxShadow: hovered ? 'var(--raise-hi)' : 'var(--raise)',
    transform: hovered ? 'translateY(-3px)' : 'translateY(0)',
    transition: 'box-shadow .35s ease, transform .35s ease',
  }
  const imgStyle: CSSProperties = {
    display: 'flex',
    alignItems: 'flex-end',
    background: 'repeating-linear-gradient(45deg, var(--img) 0 9px, var(--img2) 9px 18px)',
    flexShrink: 0,
    width: feature ? 300 : '100%',
    height: feature ? 'auto' : 150,
    minHeight: feature ? 240 : 150,
  }
  const previewStyle: CSSProperties = {
    maxHeight: hovered ? 210 : 0,
    opacity: hovered ? 1 : 0,
    overflow: 'hidden',
    transition: 'max-height .42s ease, opacity .3s ease',
  }

  return (
    <div
      style={cardStyle}
      onMouseEnter={onEnter}
      onMouseLeave={() => setHovered(false)}
    >
      {hasImg && (
        <div style={imgStyle}>
          <span
            style={{
              font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", color: 'var(--muted)',
              letterSpacing: '.05em',

              padding: '7px 9px',
            }}
          >
            {source.imgLabel}
          </span>
        </div>
      )}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', padding: '16px 17px', minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 10, marginBottom: 9 }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 7,
              font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", color: 'var(--muted)',
              letterSpacing: '.08em',
              textTransform: 'uppercase',

            }}
          >
            <span style={{ width: 5, height: 5, borderRadius: '50%', background: 'currentColor' }} />
            {source.type}
          </div>
          <button
            onClick={onToggleSave}
            aria-pressed={saved}
            aria-label={saved ? `Remove ${source.title} from board` : `Save ${source.title} to board`}
            title={saved ? 'Remove from board' : 'Save to board'}
            style={{
              position: 'relative',
              zIndex: 1,
              display: 'grid',
              placeItems: 'center',
              flexShrink: 0,
              width: 30,
              height: 30,
              borderRadius: '50%',
              background: saved ? 'var(--accent)' : 'var(--surface)',
              boxShadow: saved ? 'none' : 'var(--raise-sm)',
            }}
          >
            <Bookmark size={14} fill={saved ? '#161616' : 'none'} stroke={saved ? '#161616' : 'var(--muted)'} />
          </button>
        </div>

        <h3
          style={{
            fontFamily: "'Newsreader', serif",
            fontWeight: 500,
            fontSize: feature ? 'var(--fs-title)' : 'var(--fs-lead)',
            lineHeight: 1.22,
            letterSpacing: '-0.01em',
            margin: 0,
          }}
        >
          <a
            className="folio-card-link"
            href={source.url}
            target="_blank"
            rel="noopener noreferrer"
            style={{ color: 'inherit', textDecoration: 'none' }}
          >
            {source.title}
          </a>
        </h3>
        <div
          style={{
            font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", color: 'var(--muted)',
            letterSpacing: '.02em',

            margin: '7px 0 4px',
          }}
        >
          {source.domain}
          {source.meta ? ` · ${source.meta}` : ''}
        </div>
        <div
          data-source-url
          style={{
            position: 'relative',
            zIndex: 1,
            font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
            color: 'var(--muted)',
            overflowWrap: 'anywhere',
            display: '-webkit-box',
            WebkitBoxOrient: 'vertical',
            WebkitLineClamp: 2,
            overflow: 'hidden',
            marginBottom: 9,
          }}
        >
          {source.url}
        </div>
        <p style={{ fontSize: 'var(--fs-body)', lineHeight: 1.5, margin: 0 }}>{source.snippet}</p>

        <div style={previewStyle}>
          <div style={{ borderTop: '1px solid var(--line)', marginTop: 13, paddingTop: 13 }}>
            {(loadingPoints || (points && points.length > 0)) && (
              <div
                style={{
                  font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace", color: 'var(--muted)',
                  letterSpacing: '.1em',
                  textTransform: 'uppercase',

                  marginBottom: 8,
                }}
              >
                Key points
              </div>
            )}
            {loadingPoints && (
              <div className="folio-skel" style={{ height: 13, width: '80%', marginBottom: 8 }} />
            )}
            {points?.map((pt, i) => (
              <div
                key={i}
                style={{ display: 'flex', gap: 8, fontSize: 'var(--fs-body)', lineHeight: 1.45, marginBottom: 6 }}
              >
                <span style={{ color: 'var(--accent)' }}>·</span>
                <span>{pt}</span>
              </div>
            ))}
            <div style={{ position: 'relative', zIndex: 1, display: 'flex', gap: 18, marginTop: 6 }}>
              <button
                onClick={onRead}
                style={{ fontSize: 'var(--fs-body)', fontWeight: 600, color: 'var(--accent)' }}
              >
                Read here
              </button>
              <a
                href={source.url}
                target="_blank"
                rel="noopener noreferrer"
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 5,
                  fontSize: 'var(--fs-body)',
                  fontWeight: 600,
                  color: 'var(--accent)',
                  textDecoration: 'none',
                }}
              >
                Read source
                <ArrowRight size={12} color="currentColor" />
              </a>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
