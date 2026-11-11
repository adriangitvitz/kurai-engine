'use client'

import { AnswerCardShell } from './AnswerCardShell'
import type { InfoboxAnswer } from '@/lib/types'

export function KnowledgePanel({ infobox }: { infobox: InfoboxAnswer }) {
  return (
    <AnswerCardShell
      answerClass="Knowledge panel"
      subject={infobox.title}
      eyebrowLabel="Knowledge panel"
      source={infobox.source}
    >
      <div style={{ display: 'flex', gap: 18, alignItems: 'flex-start' }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontFamily: "'Newsreader', serif",
              fontWeight: 500,
              fontSize: 'var(--fs-title)',
              lineHeight: 1.2,
              marginBottom: 8,
            }}
          >
            {infobox.title}
          </div>
          {infobox.summary && (
            <p
              style={{
                fontFamily: "'Newsreader', serif",
                fontSize: 'var(--fs-lead)',
                lineHeight: 1.55,
                margin: '0 0 12px',
              }}
            >
              {infobox.summary}
            </p>
          )}
        </div>
        {infobox.image_url && (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={infobox.image_url}
            alt={infobox.title}
            style={{ maxWidth: 96, maxHeight: 96, borderRadius: 12, flexShrink: 0, objectFit: 'cover' }}
          />
        )}
      </div>

      {(infobox.attributes?.length ?? 0) > 0 && (
        <dl style={{ margin: '0 0 12px', display: 'grid', gridTemplateColumns: 'auto 1fr', gap: '6px 14px' }}>
          {infobox.attributes!.slice(0, 8).map((a) => (
            <div key={a.label} style={{ display: 'contents' }}>
              <dt
                style={{
                  font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
                  letterSpacing: '.06em',
                  textTransform: 'uppercase',
                  color: 'var(--muted)',
                  alignSelf: 'baseline',
                }}
              >
                {a.label}
              </dt>
              <dd style={{ margin: 0, fontSize: 'var(--fs-body)' }}>{a.value}</dd>
            </div>
          ))}
        </dl>
      )}

      {(infobox.links?.length ?? 0) > 0 && (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 14 }}>
          {infobox.links!.slice(0, 3).map((l) => (
            <a
              key={l.url}
              href={l.url}
              target="_blank"
              rel="noopener noreferrer"
              style={{
                fontSize: 'var(--fs-body)',
                color: 'var(--accent)',
                textDecoration: 'underline',
                textUnderlineOffset: 3,
              }}
            >
              {l.title}
            </a>
          ))}
        </div>
      )}
    </AnswerCardShell>
  )
}
