'use client'

import { Spark } from './icons'
import { LangToggle } from './LangToggle'
import type { SynthesizeResponse } from '@/lib/types'
import type { Language } from '@/lib/useFolioSearch'

type SynthState = 'none' | 'loading' | 'ready' | 'error'

interface AnswerZoneProps {
  synth: SynthState
  answer?: SynthesizeResponse
  synthError?: string
  sourceCount: number
  language: Language
  onSynth: () => void
  onSelectLanguage: (l: Language) => void
}

export function AnswerZone({
  synth,
  answer,
  synthError,
  sourceCount,
  language,
  onSynth,
  onSelectLanguage,
}: AnswerZoneProps) {
  if (synth === 'none') {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 22,
          flexWrap: 'wrap',
          padding: '20px 24px',
          borderRadius: 18,
          background: 'var(--surface)',
          boxShadow: 'var(--raise)',
          marginBottom: 6,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 15 }}>
          <div
            style={{
              display: 'grid',
              placeItems: 'center',
              width: 42,
              height: 42,
              borderRadius: '50%',
              background: 'var(--accent-soft)',
              flexShrink: 0,
            }}
          >
            <Spark size={19} fill="var(--accent)" />
          </div>
          <div>
            <div style={{ fontFamily: "'Newsreader', serif", fontWeight: 500, fontSize: 'var(--fs-title)', lineHeight: 1.25 }}>
              Want the short version?
            </div>
            <div style={{ fontSize: 'var(--fs-body)', marginTop: 2 }}>
              Kurai can read these {sourceCount} sources and write one answer.
            </div>
          </div>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: 9 }}>
          <LangToggle language={language} onSelect={onSelectLanguage} mini />
          <button
            onClick={onSynth}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '11px 20px',
              borderRadius: 999,
              background: 'var(--accent)',
              color: '#161616',
              fontSize: 'var(--fs-body)',
              fontWeight: 700,
              boxShadow: 'var(--raise-sm)',
            }}
          >
            <Spark size={14} fill="#161616" />
            Synthesize answer
          </button>
          <span
            style={{
              font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
              color: 'var(--muted)',
              letterSpacing: '.04em',

            }}
          >
            Uses AI · runs only on request
          </span>
        </div>
      </div>
    )
  }

  if (synth === 'loading') {
    return (
      <div style={{ marginBottom: 8 }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            marginBottom: 18,
            font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
              color: 'var(--muted)',
            letterSpacing: '.08em',
            textTransform: 'uppercase',

          }}
        >
          Reading {sourceCount} sources
          <span style={{ animation: 'folio-blink 1.2s infinite' }}>·</span>
          <span style={{ animation: 'folio-blink 1.2s infinite .2s' }}>·</span>
          <span style={{ animation: 'folio-blink 1.2s infinite .4s' }}>·</span>
        </div>
        <div className="folio-skel" style={{ height: 18, width: '96%', marginBottom: 12 }} />
        <div className="folio-skel" style={{ height: 18, width: '88%', marginBottom: 12 }} />
        <div className="folio-skel" style={{ height: 18, width: '72%' }} />
      </div>
    )
  }

  if (synth === 'error') {
    return (
      <div
        style={{
          marginBottom: 8,
          padding: '12px 16px',
          borderRadius: 14,
          background: 'var(--surface)',
          boxShadow: 'var(--raise-sm)',
          fontSize: 'var(--fs-body)',

        }}
      >
        {synthError || 'Couldn’t synthesize an answer.'}
      </div>
    )
  }

  if (!answer) return null
  const agreement = answer.agreement?.trim()
  return (
    <div style={{ marginBottom: 8 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginBottom: 18, flexWrap: 'wrap' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            padding: '6px 13px',
            borderRadius: 999,
            background: 'var(--surface)',
            boxShadow: 'var(--raise-sm)',
          }}
        >
          <span style={{ width: 7, height: 7, borderRadius: '50%', background: 'var(--accent)' }} />
          <span style={{ font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
              color: 'var(--muted)', letterSpacing: '.05em', textTransform: 'uppercase' }}>
            Synthesized · {sourceCount} sources
          </span>
        </div>
        {agreement && (
          <span style={{ font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
              color: 'var(--muted)', letterSpacing: '.04em' }}>
            {agreement}
          </span>
        )}
        <div style={{ marginLeft: 'auto' }}>
          <LangToggle language={language} onSelect={onSelectLanguage} mini />
        </div>
      </div>

      <div
        style={{
          fontFamily: "'Newsreader', serif",
          fontSize: 'var(--fs-title)',
          lineHeight: 1.62,
          maxWidth: 760,
          letterSpacing: '-0.003em',
        }}
      >
        {answer.paragraphs.map((p, i) => (
          <p key={i} style={{ margin: '0 0 19px' }}>
            {p}
          </p>
        ))}
      </div>

      {answer.facts.length > 0 && (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16, margin: '26px 0 8px' }}>
          {answer.facts.map((f, i) => (
            <div
              key={i}
              style={{
                flex: 1,
                minWidth: 170,
                padding: '18px 20px',
                borderRadius: 18,
                background: 'var(--surface)',
                boxShadow: 'var(--raise)',
              }}
            >
              <div style={{ fontFamily: "'Newsreader', serif", fontWeight: 500, fontSize: 27, lineHeight: 1, marginBottom: 8 }}>
                {f.value}
              </div>
              <div style={{ fontSize: 'var(--fs-body)', lineHeight: 1.4 }}>{f.label}</div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
