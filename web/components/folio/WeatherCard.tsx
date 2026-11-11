'use client'

import { AnswerCardShell } from './AnswerCardShell'
import type { WeatherAnswer } from '@/lib/types'

function toF(c: number): number {
  return Math.round((c * 9) / 5 + 32)
}

function formatTemps(celsius: number, countryCode?: string): string {
  const c = `${Math.round(celsius)}°C`
  return countryCode === 'US' ? `${c} / ${toF(celsius)}°F` : c
}

function formatAsOf(asOf: string, timezone?: string): string {
  try {
    const d = new Date(asOf)
    const timeText = new Intl.DateTimeFormat('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
      timeZone: timezone || undefined,
    }).format(d)
    return `As of ${timeText} (${timezone || 'UTC'})`
  } catch {
    return `As of ${asOf}`
  }
}

const dayLabel = (date: string) => {
  const d = new Date(date + 'T00:00:00')
  return Number.isNaN(d.getTime())
    ? date
    : new Intl.DateTimeFormat('en-US', { weekday: 'short' }).format(d)
}

export function WeatherCard({ weather }: { weather: WeatherAnswer }) {
  return (
    <AnswerCardShell
      answerClass="Weather"
      subject={weather.place}
      eyebrowLabel="Weather"
      source={weather.source}
      sourceHref="https://open-meteo.com/"
    >
      <div
        style={{
          fontFamily: "'Newsreader', serif",
          fontWeight: 500,
          fontSize: 'var(--fs-title)',
          lineHeight: 1.2,
          marginBottom: 8,
        }}
      >
        {weather.place}
      </div>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 12, flexWrap: 'wrap', marginBottom: 14 }}>
        <span style={{ fontFamily: "'Newsreader', serif", fontWeight: 500, fontSize: 32 }}>
          {formatTemps(weather.temperature_c, weather.country_code)}
        </span>
        {weather.condition && <span style={{ fontSize: 'var(--fs-body)' }}>{weather.condition}</span>}
      </div>

      {(weather.forecast?.length ?? 0) > 0 && (
        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: 0,
            marginBottom: 14,
          }}
        >
          {weather.forecast!.slice(0, 8).map((p, i) => (
            <div
              key={p.date}
              style={{
                padding: '6px 12px',
                borderLeft: i > 0 ? '1px solid var(--line)' : 'none',
                display: 'flex',
                flexDirection: 'column',
                gap: 3,
              }}
            >
              <span
                style={{
                  font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
                  letterSpacing: '.06em',
                  textTransform: 'uppercase',
                  color: 'var(--muted)',
                }}
              >
                {dayLabel(p.date)}
              </span>
              <span style={{ fontSize: 'var(--fs-body)' }}>
                {formatTemps(p.max_c, weather.country_code)}
                <span style={{ color: 'var(--muted)' }}> · {formatTemps(p.min_c, weather.country_code)}</span>
              </span>
            </div>
          ))}
        </div>
      )}

      <div
        data-asof={weather.as_of}
        style={{
          font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
          letterSpacing: '.08em',
          textTransform: 'uppercase',
          color: 'var(--muted)',
        }}
      >
        {formatAsOf(weather.as_of, weather.timezone)}
      </div>
    </AnswerCardShell>
  )
}
